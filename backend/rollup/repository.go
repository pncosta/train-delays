package main

import (
	"database/sql"
	"fmt"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type DBClient struct {
	dbConnectUrl string
}

func NewDBClient(dbUrl string, dbToken string) *DBClient {
	return &DBClient{
		dbConnectUrl: fmt.Sprintf("%s?authToken=%s", dbUrl, dbToken),
	}
}

var schemaStatements = []string{
	// sums and counts, never averages, so any period is an exact sum of days
	`CREATE TABLE IF NOT EXISTS daily_stats (
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
);`,
	// rank is the delay DESC, train_number ASC position within (day, service_type)
	`CREATE TABLE IF NOT EXISTS daily_worst_trips (
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
);`,
}

func (c *DBClient) InitDB() error {
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return applySchema(db)
}

func applySchema(db *sql.DB) error {
	for _, stmt := range schemaStatements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to apply schema: %w", err)
		}
	}
	return nil
}
