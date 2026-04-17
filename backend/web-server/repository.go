package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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

	// The SQL Union Query
	query := `
			SELECT service_type, COUNT(*), COALESCE(AVG(delay_minutes), 0),
			       SUM(CASE WHEN is_cancelled = 0 AND delay_minutes < 3 THEN 1 ELSE 0 END),
			       SUM(CASE WHEN is_cancelled = 0 AND delay_minutes >= 3 THEN 1 ELSE 0 END),
			       SUM(is_cancelled)
			FROM trips
			WHERE created_at >= datetime('now', '-' || ? || ' days') AND delay_minutes is not null
			GROUP BY service_type

			UNION ALL

			SELECT 'TOTAL_SYSTEM', COUNT(*), COALESCE(AVG(delay_minutes), 0),
			       SUM(CASE WHEN is_cancelled = 0 AND delay_minutes < 3 THEN 1 ELSE 0 END),
			       SUM(CASE WHEN is_cancelled = 0 AND delay_minutes >= 3 THEN 1 ELSE 0 END),
			       SUM(is_cancelled)
			FROM trips
			WHERE created_at >= datetime('now', '-' || ? || ' days')  AND delay_minutes is not null;
		`
	rows, err := db.Query(query, days, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response = &DashboardResponse{}
	response.Breakdown = []ServiceStats{}

	for rows.Next() {
		var s ServiceStats
		err := rows.Scan(
			&s.ServiceType,
			&s.TotalTrips,
			&s.AvgDelay,
			&s.OnTimeCount,
			&s.DelayedCount,
			&s.CancelledCount,
		)
		if err != nil {
			log.Printf("error scanning rows: %v", err)
			continue
		}
		response.Breakdown = append(response.Breakdown, s)
	}

	return response, nil
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
		delay_minutes, is_cancelled, created_at
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
			&t.DelayMinutes, &t.IsCancelled, &t.CreatedAt)
		if err != nil {
			return nil, err
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
		entries = append(entries, e)
	}

	return entries, nil
}
