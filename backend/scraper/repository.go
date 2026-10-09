package main

import (
	"database/sql"
	"fmt"
	"time"

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

// inserts multiple trips in the DB with the same db connection
func (c *DBClient) InsertEndingTrips(now time.Time, trips []Trip) error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	for _, trip := range trips {
		err = InsertEndingTrip(db, now, trip)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertEndingTrip inserts or merges the arrival side of one Trip.
//
// The id is normally derived from the trip's own resolved arrival day, which matches
// the departure day in the common case. But a trip whose departure and arrival are
// captured in different cron runs can have those runs land on opposite sides of
// midnight, so in the early morning we instead look up the still-open row this
// train's departure was already stored under and reuse its id - see isEarlyMorning.
func InsertEndingTrip(db *sql.DB, now time.Time, trip Trip) error {
	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}
	cancelled := trip.Supression != nil

	uid := fmt.Sprintf("%s-%d", resolveArrivalDay(trip, now), trip.TrainNumber)

	if isEarlyMorning(now) {
		openID, found, err := findOpenTripID(db, trip.TrainNumber, now)
		if err != nil {
			return err
		}
		if found {
			uid = openID
		}
	}

	query := `
		INSERT INTO trips (id, train_number, service_type,
			origin_station, destination_station,
			scheduled_arrival, actual_arrival,
			delay_minutes, is_cancelled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			actual_arrival = excluded.actual_arrival,
			scheduled_arrival = excluded.scheduled_arrival,
			delay_minutes = excluded.delay_minutes,
			is_cancelled = excluded.is_cancelled;`

	_, err := db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.ArrivalTime, trip.ETA, delay, cancelled)

	return err
}

// resolveArrivalDay returns the calendar day (2006-01-02) the trip's arrival actually
// happened on, preferring ETA over ArrivalTime to match filterEndingTrips' own
// preference. Falls back to now's day if neither clock is present or parseable.
func resolveArrivalDay(trip Trip, now time.Time) string {
	clock := trip.ArrivalTime
	if trip.ETA != nil {
		clock = trip.ETA
	}
	if clock != nil {
		if arrival, err := resolveClockTime(*clock, now); err == nil {
			return arrival.Format("2006-01-02")
		}
	}
	return now.Format("2006-01-02")
}

// findOpenTripID returns the id of the most recent trip row for trainNumber that has
// a departure but no recorded arrival yet, bounded to the last couple of days so an
// old abandoned row can't get matched to an unrelated trip reusing the same number.
func findOpenTripID(db *sql.DB, trainNumber int, now time.Time) (string, bool, error) {
	cutoff := now.Add(-48 * time.Hour).UTC().Format("2006-01-02 15:04:05")

	var id string
	err := db.QueryRow(`
		SELECT id FROM trips
		WHERE train_number = ? AND actual_arrival IS NULL AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1;`, trainNumber, cutoff).Scan(&id)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

// inserts multiple trips in the DB with the same db connection
func (c *DBClient) InsertStartingTrips(now time.Time, trips []Trip) error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	for _, trip := range trips {
		err = InsertStartingTrip(db, now, trip)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertStartingTrip inserts or merges the departure side of one Trip, keyed by the
// trip's own resolved departure day rather than a day shared across the whole run.
func InsertStartingTrip(db *sql.DB, now time.Time, trip Trip) error {
	day := now.Format("2006-01-02")
	if trip.DepartureTime != nil {
		if departure, err := resolveClockTime(*trip.DepartureTime, now); err == nil {
			day = departure.Format("2006-01-02")
		}
	}

	uid := fmt.Sprintf("%s-%d", day, trip.TrainNumber)
	query := `
		INSERT INTO trips (id, train_number, service_type,
			origin_station, destination_station,
			scheduled_departure, actual_departure,
			updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			actual_departure = excluded.actual_departure,
			scheduled_departure = excluded.scheduled_departure;`

	_, err := db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.DepartureTime, trip.ETD)

	return err
}
