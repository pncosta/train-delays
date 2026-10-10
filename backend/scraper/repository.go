package main

import (
	"database/sql"
	"fmt"
	"strings"
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
func (c *DBClient) InsertCompletedTrips(now time.Time, trips []Trip) error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer db.Close()

	for _, trip := range trips {
		err = InsertCompletedTrip(db, now, trip)
		if err != nil {
			return err
		}
	}
	return nil
}

// InsertCompletedTrip inserts or merges one Trip that already has both its departure and
// arrival side, which the per-train timetable endpoint always returns together atomically
// once a trip is complete or cancelled - unlike the old per-station flow, there's no
// scenario of two separate cron runs each writing half a trip, so there's no cross-run id
// reconciliation needed here.
func InsertCompletedTrip(db *sql.DB, now time.Time, trip Trip) error {
	delay := 0
	if trip.Delay != nil {
		delay = *trip.Delay
	}
	cancelled := trip.Supression != nil

	uid := tripID(resolveArrivalDay(trip, now), trip.TrainNumber)

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

	_, err := db.Exec(query, uid, trip.TrainNumber, trip.TrainService.Code,
		trip.TrainOrigin.Code, trip.TrainDestination.Code,
		trip.DepartureTime, trip.ArrivalTime,
		trip.ETD, trip.ETA,
		delay, cancelled)

	return err
}

// resolveArrivalDay returns the calendar day (2006-01-02) the trip's arrival actually
// happened on, preferring ETA over ArrivalTime. Falls back to now's day if neither clock
// is present or parseable.
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

// FindCapturedIDs reports which of todayIDs/yesterdayIDs already have a terminal row -
// either an arrival recorded or marked cancelled (a cancelled trip has no arrival time to
// record) - in a single batched query, one row read per already-captured id, rather than
// one query per candidate train number, since Turso bills per row read.
//
// yesterdayIDs additionally require updated_at >= recentSince: a yesterday-id match can be
// either a genuine midnight-spanning capture (always recent - it was written by a run that
// just happened) or an unrelated ~24h-old capture of the same recurring train number run
// the day before, and only the former should block polling today's actual trip. todayIDs
// need no such filter since today's own id can never be stale across a day boundary.
func (c *DBClient) FindCapturedIDs(todayIDs, yesterdayIDs []string, recentSince string) (map[string]bool, error) {
	captured := map[string]bool{}
	if len(todayIDs) == 0 && len(yesterdayIDs) == 0 {
		return captured, nil
	}

	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var conditions []string
	var args []any

	if len(todayIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("id IN (%s)", placeholders(len(todayIDs))))
		for _, id := range todayIDs {
			args = append(args, id)
		}
	}
	if len(yesterdayIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("(id IN (%s) AND updated_at >= ?)", placeholders(len(yesterdayIDs))))
		for _, id := range yesterdayIDs {
			args = append(args, id)
		}
		args = append(args, recentSince)
	}

	query := fmt.Sprintf(`
		SELECT id FROM trips
		WHERE (actual_arrival IS NOT NULL OR is_cancelled = 1) AND (%s);`, strings.Join(conditions, " OR "))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		captured[id] = true
	}
	return captured, rows.Err()
}

func placeholders(n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return strings.Join(p, ",")
}
