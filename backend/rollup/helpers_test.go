package main

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

var testLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		panic(err)
	}
	return loc
}()

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
	if err := applySchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func at(day, hour, min int) time.Time {
	return time.Date(2026, 11, day, hour, min, 0, 0, testLoc)
}

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }
