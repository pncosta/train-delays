package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite" // Pure Go driver
)

type DBClient struct {
	dbPath string
}

func newDBClient(dbPath string) *DBClient {
	return &DBClient{
		dbPath: dbPath,
	}
}

func (c *DBClient) InitDB() error {
	fmt.Printf("Initing DB in %s\n", c.dbPath)

	var err error
	db, err := sql.Open("sqlite", c.dbPath)

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

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

func (c *DBClient) InsertTrips(date string, trips []Trip, filter func(s Trip) bool) error {
	db, err := sql.Open("sqlite", c.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	_, _ = db.Exec("PRAGMA journal_mode = MEMORY;")
	_, _ = db.Exec("PRAGMA synchronous = OFF;")
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
		log.Printf("closing connection\n")
		db.Close()
	}()

	for _, trip := range trips {
		if filter(trip) {
			err = InsertTrip(tx, date, trip)
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	//	 Force FUSE to write file
	if f, ok := db.Driver().(interface{ Sync() error }); ok {
		f.Sync()
	}
	tx = nil
	return nil
}

// InsertTrip inserts one single Trip in the DB
// pass in a *sql.Tx so batch inserts can be done under same transaction
func InsertTrip(tx *sql.Tx, date string, trip Trip) error {
	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}

	cancelled := false // TODO
	uid := fmt.Sprintf("%s-%d", date, trip.TrainNumber)
	fmt.Printf("writing %s - %d %s %s %v %v %d\n", uid, trip.TrainNumber, trip.TrainOrigin.Code, trip.TrainDestination.Code, trip.ArrivalTime, trip.ETA, delay)
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
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.ArrivalTime, trip.ETA, delay, cancelled)

	return err
}
