package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// trips is owned by the scraper; the rollup never creates it outside tests.
const tripsSchema = `
CREATE TABLE IF NOT EXISTS trips (
	id TEXT PRIMARY KEY,
	train_number INTEGER,
	service_type TEXT,
	origin_station TEXT,
	destination_station TEXT,
	scheduled_departure TEXT,
	scheduled_arrival TEXT,
	actual_departure TEXT,
	actual_arrival TEXT,
	delay_minutes INTEGER,
	is_cancelled BOOLEAN,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// each connection to :memory: is its own database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(tripsSchema); err != nil {
		t.Fatal(err)
	}
	if err := initSchema(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func bp(b bool) *bool { return &b }

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

func loadStats(t *testing.T, db *sql.DB, day, svc string) statsRow {
	t.Helper()
	var r statsRow
	err := db.QueryRow(`SELECT trips, cancelled, on_time, delayed, delay_sum, delay_count, late_up_to_10, late_10_30, late_over_30, max_delay FROM daily_stats WHERE day = ? AND service_type = ?;`, day, svc).
		Scan(&r.Trips, &r.Cancelled, &r.OnTime, &r.Delayed, &r.DelaySum, &r.DelayCount, &r.L10, &r.L1030, &r.L30, &r.Max)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
