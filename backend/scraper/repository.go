package main

import (
	"database/sql"
	"fmt"

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

// InsertCompletedTrips inserts/merges each trip, keyed by its precomputed row id (see
// tripID/tripDay in train_poller.go - the poller resolves which calendar day a trip
// belongs to, since it's the one holding the wall-clock moment the trip was polled at).
func (c *DBClient) InsertCompletedTrips(trips map[string]Trip) error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	for id, trip := range trips {
		if err := InsertCompletedTrip(db, id, trip); err != nil {
			return err
		}
	}
	return nil
}

// InsertCompletedTrip inserts or merges one Trip that already has both its departure and
// arrival side under id, which the per-train timetable endpoint always returns together
// atomically once a trip is complete or cancelled - unlike the old per-station flow,
// there's no scenario of two separate cron runs each writing half a trip, so there's no
// cross-run id reconciliation needed here.
func InsertCompletedTrip(db *sql.DB, id string, trip Trip) error {
	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}
	cancelled := trip.Supression != nil

	query := `
		INSERT INTO trips (id, train_number, service_type,
			origin_station, destination_station,
			scheduled_departure, scheduled_arrival,
			actual_departure, actual_arrival,
			delay_minutes, is_cancelled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			actual_departure = excluded.actual_departure,
			scheduled_departure = excluded.scheduled_departure,
			actual_arrival = excluded.actual_arrival,
			scheduled_arrival = excluded.scheduled_arrival,
			delay_minutes = excluded.delay_minutes,
			is_cancelled = excluded.is_cancelled;`

	_, err := db.Exec(query, id, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.DepartureTime, trip.ArrivalTime,
		trip.ETD, trip.ETA,
		delay, cancelled)

	return err
}
