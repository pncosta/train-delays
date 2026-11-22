package main

import (
	"context"
	"database/sql"
	"fmt"
	"train-delays/shared"

	_ "github.com/tursodatabase/libsql-client-go/libsql" // New driver
)

type DBClient struct {
	dbUrl        string
	dbConnectUrl string
	dbToken      string
}

func NewDBClient(dbUrl string, dbToken string) *DBClient {
	return &DBClient{
		dbUrl:        dbUrl,
		dbToken:      dbToken,
		dbConnectUrl: fmt.Sprintf("%s?authToken=%s", dbUrl, dbToken),
	}
}

func (c *DBClient) InitDB() error {
	var err error
	db, err := sql.Open("libsql", c.dbConnectUrl) // TODO: make this connection persistent instead of opening/closing for each query

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()
	return nil
}

// inserts multiple trips in the DB with the same db connection
func (c *DBClient) GetSummary() (*DashboardResponse, error) {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	days := 7 // TODO make parameter

	groups, err := selectSummaryGroups(db, days)
	if err != nil {
		return nil, err
	}
	return summarizeGroups(groups), nil
}

func selectSummaryGroups(db *sql.DB, days int) ([]summaryGroup, error) {
	rows, err := db.Query(`
		SELECT COALESCE(service_type, ''), delay_minutes, COALESCE(is_cancelled, 0), scheduled_arrival IS NOT NULL, COUNT(*)
		FROM trips
		WHERE created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY service_type, delay_minutes, is_cancelled, scheduled_arrival IS NOT NULL;`, days)
	if err != nil {
		return nil, fmt.Errorf("failed to select summary groups: %w", err)
	}
	defer rows.Close()

	var groups []summaryGroup
	for rows.Next() {
		var group summaryGroup
		err := rows.Scan(&group.ServiceType, &group.DelayMinutes, &group.IsCancelled, &group.ArrivalSeen, &group.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary group: %w", err)
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func selectDailyStats(ctx context.Context, db *sql.DB, fromDay, toDay string) ([]DailyStats, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT day, service_type, trips, cancelled, on_time, delayed, delay_sum, delay_count,
		       late_up_to_10, late_10_30, late_over_30, max_delay
		FROM daily_stats
		WHERE day >= ? AND day <= ?;`, fromDay, toDay)
	if err != nil {
		return nil, fmt.Errorf("failed to select daily_stats %s..%s: %w", fromDay, toDay, err)
	}
	defer rows.Close()

	var stats []DailyStats
	for rows.Next() {
		var row DailyStats
		err := rows.Scan(&row.Day, &row.ServiceType, &row.Trips, &row.Cancelled, &row.OnTime, &row.Delayed,
			&row.DelaySum, &row.DelayCount, &row.LateUpTo10, &row.Late10To30, &row.LateOver30, &row.MaxDelay)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily_stats: %w", err)
		}
		stats = append(stats, row)
	}
	return stats, rows.Err()
}

func selectDailyWorstTrips(ctx context.Context, db *sql.DB, fromDay, toDay string) ([]DailyWorstTrip, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT day, service_type, trip_id, COALESCE(train_number, 0), COALESCE(origin_station, ''),
		       COALESCE(destination_station, ''), COALESCE(scheduled_arrival, ''), COALESCE(actual_arrival, ''),
		       delay_minutes
		FROM daily_worst_trips
		WHERE day >= ? AND day <= ?;`, fromDay, toDay)
	if err != nil {
		return nil, fmt.Errorf("failed to select daily_worst_trips %s..%s: %w", fromDay, toDay, err)
	}
	defer rows.Close()

	var trips []DailyWorstTrip
	for rows.Next() {
		var trip DailyWorstTrip
		err := rows.Scan(&trip.Day, &trip.ServiceType, &trip.TripID, &trip.TrainNumber,
			&trip.OriginStation, &trip.DestinationStation, &trip.ScheduledArrival, &trip.ActualArrival,
			&trip.DelayMinutes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily_worst_trips: %w", err)
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}

func (c *DBClient) GetWorstlDelays(ctx context.Context, days int, limit int) ([]Trip, error) {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
        SELECT id, train_number, service_type, 
		origin_station, destination_station, 
		scheduled_arrival, 
		scheduled_departure, 
		actual_departure, 
		actual_arrival, 
		delay_minutes, is_cancelled, created_at, updated_at
        FROM trips
        WHERE created_at >= datetime('now', '-' || ? || ' days')
          AND delay_minutes IS NOT NULL
          AND is_cancelled = 0
        ORDER BY delay_minutes DESC
        LIMIT ?;
    `
	rows, err := db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []Trip
	for rows.Next() {
		var t Trip
		err := rows.Scan(&t.Id, &t.TrainNumber, &t.ServiceType,
			&t.OriginStation, &t.DestinationStation,
			&t.ScheduledArrival, &t.ScheduledDeparture, &t.ActualDeparture, &t.ActualArrival,
			&t.DelayMinutes, &t.IsCancelled, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		originStation, ok := shared.Stations[t.OriginStation]
		if ok {
			t.OriginStationName = originStation.Designation
		}

		destinationStation, ok := shared.Stations[t.DestinationStation]
		if ok {
			t.DestinationStationName = destinationStation.Designation
		}
		trips = append(trips, t)
	}
	return trips, nil
}

func (c *DBClient) GetCancellations(ctx context.Context, days int, limit int) ([]Trip, error) {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := `
        SELECT id, train_number, service_type, 
		origin_station, destination_station, 
		scheduled_arrival, 
		scheduled_departure, 
		actual_departure, 
		actual_arrival, 
		delay_minutes, is_cancelled, created_at, updated_at
        FROM trips
        WHERE created_at >= datetime('now', '-' || ? || ' days')
          AND is_cancelled = 1
        ORDER BY created_at DESC
        LIMIT ?;
    `
	rows, err := db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []Trip
	for rows.Next() {
		var t Trip
		err := rows.Scan(&t.Id, &t.TrainNumber, &t.ServiceType,
			&t.OriginStation, &t.DestinationStation,
			&t.ScheduledArrival, &t.ScheduledDeparture, &t.ActualDeparture, &t.ActualArrival,
			&t.DelayMinutes, &t.IsCancelled, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}

		originStation, ok := shared.Stations[t.OriginStation]
		if ok {
			t.OriginStationName = originStation.Designation
		}

		destinationStation, ok := shared.Stations[t.DestinationStation]
		if ok {
			t.DestinationStationName = destinationStation.Designation
		}

		trips = append(trips, t)
	}
	return trips, nil
}

func (c *DBClient) GetWorstAverageDelays(ctx context.Context, days int, limit int) ([]LeaderboardEntry, error) {

	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	// We group by all descriptive fields to ensure each "Route" is treated uniquely.
	// This prevents the DB from picking a random station name for a train number.
	query := `
        SELECT 
            train_number, 
            service_type,
            origin_station, 
            destination_station, 
            AVG(delay_minutes) as avg_delay, 
            COUNT(*) as total_trips
        FROM trips
        WHERE created_at >= datetime('now', '-' || ? || ' days')
          AND is_cancelled = 0
          AND delay_minutes IS NOT NULL
        GROUP BY 
            train_number, 
            service_type, 
            origin_station, 
            destination_station
        HAVING total_trips > 3
        ORDER BY avg_delay DESC
        LIMIT ?;
    `

	rows, err := db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		err := rows.Scan(
			&e.TrainNumber,
			&e.ServiceType,
			&e.OriginStation,
			&e.DestinationStation,
			&e.Value,
			&e.Count,
		)
		if err != nil {
			return nil, err
		}
		originStation, ok := shared.Stations[e.OriginStation]
		if ok {
			e.OriginStationName = originStation.Designation
		}

		destinationStation, ok := shared.Stations[e.DestinationStation]
		if ok {
			e.DestinationStationName = destinationStation.Designation
		}

		entries = append(entries, e)
	}

	return entries, nil
}
