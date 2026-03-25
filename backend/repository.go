package main

import (
	"database/sql"
	"fmt"

	// _ "modernc.org/sqlite" // Pure Go driver
	_ "github.com/tursodatabase/libsql-client-go/libsql" // New driver
)

type DBClient struct {
	dbUrl        string
	dbConnectUrl string
	dbToken      string
}

func newDBClient(dbUrl string, dbToken string) *DBClient {
	return &DBClient{
		dbUrl:        dbUrl,
		dbToken:      dbToken,
		dbConnectUrl: fmt.Sprintf("%s?authToken=%s", dbUrl, dbToken),
	}
}

func (c *DBClient) InitDB() error {
	var err error
	db, err := sql.Open("libsql", c.dbConnectUrl)

	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	// Create the schema if it doesn't exist
	schema := `
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

	_, err = db.Exec(schema)
	return err
}

// inserts multiple trips in the DB with the same db connection
func (c *DBClient) InsertTrips(date string, trips []Trip, filter func(s Trip) bool) error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	for _, trip := range trips {
		if filter(trip) {
			err = InsertTrip(db, date, trip)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// InsertTrip inserts one single Trip in the DB
// pass in a *sql.Tx so batch inserts can be done under same transaction
func InsertTrip(db *sql.DB, date string, trip Trip) error {
	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}

	cancelled := trip.Supression != nil
	uid := fmt.Sprintf("%s-%d", date, trip.TrainNumber)
	query := `
		INSERT INTO trips (id, train_number, service_type, 
			origin_station, destination_station,  
			scheduled_departure, actual_departure,  
			scheduled_arrival, actual_arrival, 
			delay_minutes, is_cancelled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			actual_arrival = excluded.actual_arrival,
			actual_departure = excluded.actual_departure,
			delay_minutes = excluded.delay_minutes,
			is_cancelled = excluded.is_cancelled;`

	_, err := db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.DepartureTime, trip.ETD,
		trip.ArrivalTime, trip.ETA, delay, cancelled)

	return err
}
