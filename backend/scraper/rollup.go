package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const dayLayout = "2006-01-02"

// The overnight gap (last arrival ~02:52, first departure ~04:05) means yesterday is complete from 04:00.
const rollupHour = 4

// Worst trips are the delayed subset, so the list matches the delayed counter.
// Delay-eligible rows have an arrival side and aren't cancelled; a NULL delay counts as 0.
// Delayed is > 3 min for U and > 5 min for every other type; bands partition on that threshold.
// Ids are YYYY-MM-DD-<train>, so the id range filter hits the PK index and day is the id prefix.
const rollupBase = `
WITH t AS (
	SELECT
		substr(id, 1, 10) AS day,
		COALESCE(service_type, '') AS service_type,
		id,
		train_number,
		origin_station,
		destination_station,
		scheduled_arrival,
		actual_arrival,
		COALESCE(is_cancelled, 0) AS cancelled,
		scheduled_arrival IS NOT NULL AND COALESCE(is_cancelled, 0) = 0 AS eligible,
		COALESCE(delay_minutes, 0) AS d,
		CASE WHEN service_type = 'U' THEN 3 ELSE 5 END AS thr
	FROM trips
	WHERE id >= ? AND id < ?
)`

const rollupStatsSQL = rollupBase + `
INSERT INTO daily_stats (day, service_type, trips, cancelled, on_time, delayed, delay_sum, delay_count,
	late_up_to_10, late_10_30, late_over_30, max_delay)
SELECT
	day,
	service_type,
	COUNT(*),
	SUM(cancelled),
	COALESCE(SUM(eligible AND d <= thr), 0),
	COALESCE(SUM(eligible AND d > thr), 0),
	COALESCE(SUM(CASE WHEN eligible THEN d END), 0),
	COALESCE(SUM(eligible), 0),
	COALESCE(SUM(eligible AND d > thr AND d <= 10), 0),
	COALESCE(SUM(eligible AND d > 10 AND d <= 30), 0),
	COALESCE(SUM(eligible AND d > 30), 0),
	COALESCE(MAX(CASE WHEN eligible THEN d END), 0)
FROM t
GROUP BY day, service_type;`

const rollupWorstSQL = rollupBase + `
INSERT INTO daily_worst_trips (day, service_type, rank, trip_id, train_number, origin_station,
	destination_station, scheduled_arrival, actual_arrival, delay_minutes)
SELECT day, service_type, rank, id, train_number, origin_station, destination_station,
	scheduled_arrival, actual_arrival, d
FROM (
	SELECT *, ROW_NUMBER() OVER (PARTITION BY day, service_type ORDER BY d DESC, train_number ASC, id ASC) AS rank
	FROM t
	WHERE eligible AND d > thr
)
WHERE rank <= 10;`

// yesterdayToRollup returns the day to roll up once Lisbon time is past the overnight gap.
func yesterdayToRollup(nowLisbon time.Time) (string, bool) {
	if nowLisbon.Hour() < rollupHour {
		return "", false
	}
	return nowLisbon.AddDate(0, 0, -1).Format(dayLayout), true
}

// rollupRange replaces the rollup rows for days in [from, to) atomically; from and to are YYYY-MM-DD.
func rollupRange(db *sql.DB, from, to string) error {
	var marker int
	err := db.QueryRow(`SELECT 1 FROM trips WHERE id >= ? AND id < ? LIMIT 1;`, from, to).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check trips for %s..%s: %w", from, to, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin rollup: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{"daily_stats", "daily_worst_trips"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE day >= ? AND day < ?;`, from, to); err != nil {
			return fmt.Errorf("failed to clear %s: %w", table, err)
		}
	}
	if _, err := tx.Exec(rollupStatsSQL, from, to); err != nil {
		return fmt.Errorf("failed to roll up daily_stats: %w", err)
	}
	if _, err := tx.Exec(rollupWorstSQL, from, to); err != nil {
		return fmt.Errorf("failed to roll up daily_worst_trips: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollup: %w", err)
	}
	return nil
}

func rollupDay(db *sql.DB, day string) error {
	d, err := time.Parse(dayLayout, day)
	if err != nil {
		return fmt.Errorf("failed to parse day %q: %w", day, err)
	}
	return rollupRange(db, day, d.AddDate(0, 0, 1).Format(dayLayout))
}

// rollupYesterdayIfDue costs one marker read when yesterday is already rolled up.
func rollupYesterdayIfDue(db *sql.DB, nowLisbon time.Time) error {
	day, due := yesterdayToRollup(nowLisbon)
	if !due {
		return nil
	}
	var marker int
	err := db.QueryRow(`SELECT 1 FROM daily_stats WHERE day = ? LIMIT 1;`, day).Scan(&marker)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to read rollup marker: %w", err)
	}
	return rollupDay(db, day)
}

// backfillRollups rolls up every complete day in trips (yesterday only from rollupHour), one calendar
// month per transaction to keep each Turso transaction bounded.
func backfillRollups(db *sql.DB, nowLisbon time.Time) error {
	var minID sql.NullString
	if err := db.QueryRow(`SELECT MIN(id) FROM trips;`).Scan(&minID); err != nil {
		return fmt.Errorf("failed to find first trip day: %w", err)
	}
	if !minID.Valid || len(minID.String) < 10 {
		return nil
	}
	first, err := time.Parse(dayLayout, minID.String[:10])
	if err != nil {
		return fmt.Errorf("failed to parse first trip day %q: %w", minID.String, err)
	}

	end := nowLisbon
	if _, due := yesterdayToRollup(nowLisbon); !due {
		end = nowLisbon.AddDate(0, 0, -1)
	}
	until := end.Format(dayLayout)
	for start := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC); start.Format(dayLayout) < until; start = start.AddDate(0, 1, 0) {
		from := start.Format(dayLayout)
		if from < minID.String[:10] {
			from = minID.String[:10]
		}
		to := start.AddDate(0, 1, 0).Format(dayLayout)
		if to > until {
			to = until
		}
		if err := rollupRange(db, from, to); err != nil {
			return fmt.Errorf("backfill %s..%s: %w", from, to, err)
		}
	}
	return nil
}

func lisbonNow(t time.Time) (time.Time, error) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load Europe/Lisbon: %w", err)
	}
	return t.In(lisbon), nil
}

func (c *DBClient) RollupYesterdayIfDue(t time.Time) error {
	nowLisbon, err := lisbonNow(t)
	if err != nil {
		return err
	}
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return rollupYesterdayIfDue(db, nowLisbon)
}

func (c *DBClient) BackfillRollups(t time.Time) error {
	nowLisbon, err := lisbonNow(t)
	if err != nil {
		return err
	}
	db, err := sql.Open("libsql", c.dbConnectUrl)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	return backfillRollups(db, nowLisbon)
}
