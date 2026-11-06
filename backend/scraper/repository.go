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
// the departure day in the common case. An early-morning arrival from a trip that
// crossed midnight belongs to the previous day's row instead - see findPreviousDayTripID.
func InsertEndingTrip(db *sql.DB, now time.Time, trip Trip) error {
	cancelled := trip.Supression != nil

	uid := fmt.Sprintf("%s-%d", resolveArrivalDay(trip, now), trip.TrainNumber)

	prevID, found, err := findPreviousDayTripID(db, trip, now)
	if err != nil {
		return err
	}
	if found {
		uid = prevID
	}

	// CP stops sending Delay once the train arrives; a nil Delay must not reset the stored one.
	query := `
		INSERT INTO trips (id, train_number, service_type,
			origin_station, destination_station,
			scheduled_arrival, actual_arrival,
			delay_minutes, is_cancelled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, 0), ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			actual_arrival = COALESCE(excluded.actual_arrival, trips.actual_arrival),
			scheduled_arrival = COALESCE(excluded.scheduled_arrival, trips.scheduled_arrival),
			delay_minutes = COALESCE(?, trips.delay_minutes),
			is_cancelled = excluded.is_cancelled;`

	_, err = db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.ArrivalTime, trip.ETA, trip.Delay, cancelled, trip.Delay)

	return err
}

// resolveArrivalDay returns the calendar day (2006-01-02) the trip's arrival actually
// happened on. Falls back to now's day if there is no parseable arrival clock.
func resolveArrivalDay(trip Trip, now time.Time) string {
	if clock := arrivalClock(trip); clock != nil {
		if arrival, err := resolveClockTime(*clock, now); err == nil {
			return arrival.Format("2006-01-02")
		}
	}
	return now.Format("2006-01-02")
}

// findPreviousDayTripID returns the id of the row for the day before the trip's
// resolved arrival day when that row is a midnight-crossing trip this arrival
// completes. The row is looked up by id (a train runs at most once per 24h) rather than
// by open/created_at state: rows are created early on their own day and keep receiving
// arrival updates after midnight. Gated on the arrival clock, not now, because the
// ending-trip window straddles midnight and the early-morning boundary.
func findPreviousDayTripID(db *sql.DB, trip Trip, now time.Time) (string, bool, error) {
	clock := arrivalClock(trip)
	if clock == nil {
		return "", false, nil
	}
	arrival, err := resolveClockTime(*clock, now)
	if err != nil || !isEarlyMorning(arrival) {
		return "", false, nil
	}

	id := fmt.Sprintf("%s-%d", arrival.AddDate(0, 0, -1).Format("2006-01-02"), trip.TrainNumber)

	var departure sql.NullString
	err = db.QueryRow(`SELECT scheduled_departure FROM trips WHERE id = ?;`, id).Scan(&departure)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !departure.Valid || !crossesMidnight(departure.String, *clock) {
		return "", false, nil
	}
	return id, true, nil
}

// crossesMidnight reports whether arriving at the arrival clock after departing at the
// departure clock must have wrapped past midnight. Zero-padded HH:MM compares lexically.
func crossesMidnight(departure, arrival string) bool {
	return departure > arrival
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
			actual_departure = COALESCE(excluded.actual_departure, trips.actual_departure),
			scheduled_departure = COALESCE(excluded.scheduled_departure, trips.scheduled_departure);`

	_, err := db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.DepartureTime, trip.ETD)

	return err
}
