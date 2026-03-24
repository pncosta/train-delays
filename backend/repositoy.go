package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite" // Pure Go driver
)

var db *sql.DB

func InitDB() error {
	// Default to local file, but override in Cloud Run to "/data/trains.db"
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/trains_local.db"
	}

	fmt.Printf("Initing DB in %s\n", dbPath)

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// set SQLite to use memory for its journal (no .db-journal file created)
	_, err = db.Exec("PRAGMA journal_mode=MEMORY;")
	if err != nil {
		return err
	}

	// Reduce the "synchronous" level to be more FUSE-friendly
	_, err = db.Exec("PRAGMA synchronous=OFF;")
	if err != nil {
		return err
	}
	// Create the schema if it doesn't exist
	schema := `
	CREATE TABLE IF NOT EXISTS trips (
		id TEXT PRIMARY KEY,
		train_number INTEGER,
		service_type TEXT,
		origin_station TEXT,
		destination_station TEXT,
		scheduled_arrival TEXT,
		actual_arrival TEXT,
		delay_minutes INTEGER,
		is_cancelled BOOLEAN,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(schema)
	return err
}

// SaveSingleArrival does the work for just one train.
// It requires a transaction (*sql.Tx) to be passed in.
func SaveTrip(tx *sql.Tx, date string, trip Trip) error {

	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}

	cancelled := false // TODO
	uid := fmt.Sprintf("%s-%d", date, trip.TrainNumber)

	query := `
		INSERT INTO trips (id, train_number, service_type, origin_station, 
			destination_station, scheduled_arrival, actual_arrival, 
			delay_minutes, is_cancelled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			actual_arrival = excluded.actual_arrival,
			delay_minutes = excluded.delay_minutes,
			is_cancelled = excluded.is_cancelled;`

	_, err := tx.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Designation, trip.TrainDestination.Designation,
		trip.ArrivalTime, trip.ETA, delay, cancelled)

	return err
}
