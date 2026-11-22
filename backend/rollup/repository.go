package main

import (
	"context"
	"database/sql"
	"fmt"
)

func initSchema(ctx context.Context, db *sql.DB) error {
	// sums and counts, never averages, so any period is an exact sum of days
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS daily_stats (
	day TEXT NOT NULL,
	service_type TEXT NOT NULL,
	trips INTEGER NOT NULL,
	cancelled INTEGER NOT NULL,
	on_time INTEGER NOT NULL,
	delayed INTEGER NOT NULL,
	delay_sum INTEGER NOT NULL,
	delay_count INTEGER NOT NULL,
	late_up_to_10 INTEGER NOT NULL,
	late_10_30 INTEGER NOT NULL,
	late_over_30 INTEGER NOT NULL,
	max_delay INTEGER NOT NULL,
	PRIMARY KEY (day, service_type)
);`)
	if err != nil {
		return fmt.Errorf("failed to create daily_stats: %w", err)
	}
	// rank is the delay DESC, train_number ASC position within (day, service_type)
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS daily_worst_trips (
	day TEXT NOT NULL,
	service_type TEXT NOT NULL,
	rank INTEGER NOT NULL,
	trip_id TEXT NOT NULL,
	train_number INTEGER,
	origin_station TEXT,
	destination_station TEXT,
	scheduled_arrival TEXT,
	actual_arrival TEXT,
	delay_minutes INTEGER NOT NULL,
	PRIMARY KEY (day, service_type, rank)
);`)
	if err != nil {
		return fmt.Errorf("failed to create daily_worst_trips: %w", err)
	}
	return nil
}

// selectTrips returns trips with id in [from, to).
func selectTrips(ctx context.Context, db *sql.DB, from, to string) ([]trip, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, train_number, service_type, origin_station, destination_station,
		scheduled_arrival, actual_arrival, delay_minutes, is_cancelled
		FROM trips WHERE id >= ? AND id < ?;`, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to select trips for %s..%s: %w", from, to, err)
	}
	defer rows.Close()

	var trips []trip
	for rows.Next() {
		var t trip
		if err := rows.Scan(&t.id, &t.trainNumber, &t.serviceType, &t.originStation, &t.destinationStation,
			&t.scheduledArrival, &t.actualArrival, &t.delayMinutes, &t.isCancelled); err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}
		trips = append(trips, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read trips for %s..%s: %w", from, to, err)
	}
	return trips, nil
}

// selectMinTripID reports false when trips is empty.
func selectMinTripID(ctx context.Context, db *sql.DB) (string, bool, error) {
	var minID sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT MIN(id) FROM trips;`).Scan(&minID); err != nil {
		return "", false, fmt.Errorf("failed to select min trip id: %w", err)
	}
	return minID.String, minID.Valid, nil
}

// insertRollups writes a range's stats and worst trips in one transaction; rows that already exist are kept.
func insertRollups(ctx context.Context, db *sql.DB, stats []dailyStats, worst []worstTrip) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin rollup: %w", err)
	}
	defer tx.Rollback()

	for _, s := range stats {
		if err := insertDailyStats(ctx, tx, s); err != nil {
			return fmt.Errorf("failed to insert daily_stats %s/%q: %w", s.day, s.serviceType, err)
		}
	}
	for _, w := range worst {
		if err := insertWorstTrip(ctx, tx, w); err != nil {
			return fmt.Errorf("failed to insert daily_worst_trips %s/%q #%d: %w", w.day, w.serviceType, w.rank, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollup: %w", err)
	}
	return nil
}

func insertDailyStats(ctx context.Context, tx *sql.Tx, s dailyStats) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO daily_stats (day, service_type, trips, cancelled, on_time, delayed, delay_sum,
		delay_count, late_up_to_10, late_10_30, late_over_30, max_delay)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING;`,
		s.day, s.serviceType, s.trips, s.cancelled, s.onTime, s.delayed, s.delaySum,
		s.delayCount, s.lateUpTo10, s.late10To30, s.lateOver30, s.maxDelay)
	return err
}

func insertWorstTrip(ctx context.Context, tx *sql.Tx, w worstTrip) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO daily_worst_trips (day, service_type, rank, trip_id, train_number,
		origin_station, destination_station, scheduled_arrival, actual_arrival, delay_minutes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING;`,
		w.day, w.serviceType, w.rank, w.trip.id, w.trip.trainNumber, w.trip.originStation,
		w.trip.destinationStation, w.trip.scheduledArrival, w.trip.actualArrival, w.delay)
	return err
}
