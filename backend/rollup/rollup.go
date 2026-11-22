package main

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"
)

const dayLayout = "2006-01-02"

// Delayed is > urbanDelayedMinutes for urban trips and > defaultDelayedMinutes otherwise; late bands partition on that threshold.
const (
	urbanServiceType      = "U"
	urbanDelayedMinutes   = 3
	defaultDelayedMinutes = 5
	lateBandShortMax      = 10
	lateBandMediumMax     = 30
)

// The overnight gap (last arrival ~02:52, first departure ~04:05) means yesterday is complete from 04:00.
const rollupHour = 4

const worstTripsPerGroup = 10

type tripRow struct {
	id                 string
	trainNumber        sql.NullInt64
	serviceType        sql.NullString
	originStation      sql.NullString
	destinationStation sql.NullString
	scheduledArrival   sql.NullString
	actualArrival      sql.NullString
	delayMinutes       sql.NullInt64
	isCancelled        sql.NullBool
}

type dayService struct {
	day         string
	serviceType string
}

type dailyStats struct {
	dayService
	trips, cancelled, onTime, delayed, delaySum, delayCount int
	lateUpTo10, late10To30, lateOver30, maxDelay            int
}

type worstTrip struct {
	dayService
	rank int
	trip tripRow
}

// The id prefix is the day (ids are YYYY-MM-DD-<train>).
func (t tripRow) day() string { return t.id[:10] }

func (t tripRow) service() string { return t.serviceType.String }

func (t tripRow) group() dayService { return dayService{t.day(), t.service()} }

// Delay-eligible rows have an arrival side and aren't cancelled.
func (t tripRow) eligible() bool { return t.scheduledArrival.Valid && !t.isCancelled.Bool }

// A NULL delay counts as 0.
func (t tripRow) delay() int { return int(t.delayMinutes.Int64) }

func delayedThreshold(serviceType string) int {
	if serviceType == urbanServiceType {
		return urbanDelayedMinutes
	}
	return defaultDelayedMinutes
}

func sortGroups(keys []dayService) {
	slices.SortFunc(keys, func(a, b dayService) int {
		return cmp.Or(cmp.Compare(a.day, b.day), cmp.Compare(a.serviceType, b.serviceType))
	})
}

func buildDailyStats(trips []tripRow) []dailyStats {
	byGroup := map[dayService]*dailyStats{}
	for _, t := range trips {
		g := t.group()
		s := byGroup[g]
		if s == nil {
			s = &dailyStats{dayService: g}
			byGroup[g] = s
		}
		s.trips++
		if t.isCancelled.Bool {
			s.cancelled++
		}
		if !t.eligible() {
			continue
		}
		d := t.delay()
		if s.delayCount == 0 {
			s.maxDelay = d
		}
		s.maxDelay = max(s.maxDelay, d)
		s.delayCount++
		s.delaySum += d
		if d <= delayedThreshold(g.serviceType) {
			s.onTime++
			continue
		}
		s.delayed++
		switch {
		case d <= lateBandShortMax:
			s.lateUpTo10++
		case d <= lateBandMediumMax:
			s.late10To30++
		default:
			s.lateOver30++
		}
	}
	keys := make([]dayService, 0, len(byGroup))
	for g := range byGroup {
		keys = append(keys, g)
	}
	sortGroups(keys)
	out := make([]dailyStats, 0, len(keys))
	for _, g := range keys {
		out = append(out, *byGroup[g])
	}
	return out
}

// Worst trips are the delayed subset, so the list matches the delayed counter.
func buildWorstTrips(trips []tripRow) []worstTrip {
	byGroup := map[dayService][]tripRow{}
	for _, t := range trips {
		if t.eligible() && t.delay() > delayedThreshold(t.service()) {
			byGroup[t.group()] = append(byGroup[t.group()], t)
		}
	}
	keys := make([]dayService, 0, len(byGroup))
	for g := range byGroup {
		keys = append(keys, g)
	}
	sortGroups(keys)
	var out []worstTrip
	for _, g := range keys {
		group := byGroup[g]
		slices.SortFunc(group, compareWorse)
		for i, t := range group[:min(len(group), worstTripsPerGroup)] {
			out = append(out, worstTrip{dayService: g, rank: i + 1, trip: t})
		}
	}
	return out
}

// Delay DESC, train_number ASC (NULL first, as in SQL), id ASC.
func compareWorse(a, b tripRow) int {
	if c := cmp.Compare(b.delay(), a.delay()); c != 0 {
		return c
	}
	if a.trainNumber.Valid != b.trainNumber.Valid {
		if !a.trainNumber.Valid {
			return -1
		}
		return 1
	}
	return cmp.Or(cmp.Compare(a.trainNumber.Int64, b.trainNumber.Int64), cmp.Compare(a.id, b.id))
}

func selectTrips(db *sql.DB, from, to string) ([]tripRow, error) {
	rows, err := db.Query(`SELECT id, train_number, service_type, origin_station, destination_station,
		scheduled_arrival, actual_arrival, delay_minutes, is_cancelled
		FROM trips WHERE id >= ? AND id < ?;`, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to select trips for %s..%s: %w", from, to, err)
	}
	defer rows.Close()

	var trips []tripRow
	for rows.Next() {
		var t tripRow
		if err := rows.Scan(&t.id, &t.trainNumber, &t.serviceType, &t.originStation, &t.destinationStation,
			&t.scheduledArrival, &t.actualArrival, &t.delayMinutes, &t.isCancelled); err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}
		if len(t.id) < len(dayLayout) {
			log.Printf("skipping trip with malformed id %q", t.id)
			continue
		}
		trips = append(trips, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read trips for %s..%s: %w", from, to, err)
	}
	return trips, nil
}

func dayRolledUp(db *sql.DB, day string) (bool, error) {
	var marker int
	err := db.QueryRow(`SELECT 1 FROM daily_stats WHERE day = ? LIMIT 1;`, day).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func selectMinTripID(db *sql.DB) (sql.NullString, error) {
	var minID sql.NullString
	err := db.QueryRow(`SELECT MIN(id) FROM trips;`).Scan(&minID)
	return minID, err
}

func deleteDailyStats(tx *sql.Tx, from, to string) error {
	_, err := tx.Exec(`DELETE FROM daily_stats WHERE day >= ? AND day < ?;`, from, to)
	return err
}

func deleteWorstTrips(tx *sql.Tx, from, to string) error {
	_, err := tx.Exec(`DELETE FROM daily_worst_trips WHERE day >= ? AND day < ?;`, from, to)
	return err
}

func insertDailyStats(tx *sql.Tx, s dailyStats) error {
	_, err := tx.Exec(`INSERT INTO daily_stats (day, service_type, trips, cancelled, on_time, delayed, delay_sum,
		delay_count, late_up_to_10, late_10_30, late_over_30, max_delay)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		s.day, s.serviceType, s.trips, s.cancelled, s.onTime, s.delayed, s.delaySum,
		s.delayCount, s.lateUpTo10, s.late10To30, s.lateOver30, s.maxDelay)
	return err
}

func insertWorstTrip(tx *sql.Tx, w worstTrip) error {
	_, err := tx.Exec(`INSERT INTO daily_worst_trips (day, service_type, rank, trip_id, train_number,
		origin_station, destination_station, scheduled_arrival, actual_arrival, delay_minutes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		w.day, w.serviceType, w.rank, w.trip.id, w.trip.trainNumber, w.trip.originStation,
		w.trip.destinationStation, w.trip.scheduledArrival, w.trip.actualArrival, w.trip.delay())
	return err
}

// yesterdayToRollup returns the day to roll up once Lisbon time is past the overnight gap.
func yesterdayToRollup(nowLisbon time.Time) (string, bool) {
	if nowLisbon.Hour() < rollupHour {
		return "", false
	}
	return nowLisbon.AddDate(0, 0, -1).Format(dayLayout), true
}

// rollupRange replaces the rollup rows for days in [from, to) atomically; from and to are YYYY-MM-DD.
func rollupRange(db *sql.DB, from, to string) error {
	trips, err := selectTrips(db, from, to)
	if err != nil {
		return err
	}
	if len(trips) == 0 {
		return nil
	}
	stats := buildDailyStats(trips)
	worst := buildWorstTrips(trips)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin rollup: %w", err)
	}
	defer tx.Rollback()

	if err := deleteDailyStats(tx, from, to); err != nil {
		return fmt.Errorf("failed to clear daily_stats: %w", err)
	}
	if err := deleteWorstTrips(tx, from, to); err != nil {
		return fmt.Errorf("failed to clear daily_worst_trips: %w", err)
	}
	for _, s := range stats {
		if err := insertDailyStats(tx, s); err != nil {
			return fmt.Errorf("failed to insert daily_stats %s/%q: %w", s.day, s.serviceType, err)
		}
	}
	for _, w := range worst {
		if err := insertWorstTrip(tx, w); err != nil {
			return fmt.Errorf("failed to insert daily_worst_trips %s/%q #%d: %w", w.day, w.serviceType, w.rank, err)
		}
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
	done, err := dayRolledUp(db, day)
	if err != nil {
		return fmt.Errorf("failed to read rollup marker: %w", err)
	}
	if done {
		return nil
	}
	return rollupDay(db, day)
}

// backfillRollups rolls up every complete day in trips (yesterday only from rollupHour), one calendar
// month per transaction to keep each Turso transaction bounded.
func backfillRollups(db *sql.DB, nowLisbon time.Time) error {
	minID, err := selectMinTripID(db)
	if err != nil {
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
