package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

type seedTrip struct {
	id, svc   string
	train     int
	schedArr  *string
	delay     *int
	cancelled any
}

func seed(t *testing.T, db *sql.DB, trips ...seedTrip) {
	t.Helper()
	for _, tr := range trips {
		_, err := db.Exec(`INSERT INTO trips (id, train_number, service_type, origin_station, destination_station, scheduled_arrival, delay_minutes, is_cancelled) VALUES (?, ?, ?, 'A', 'B', ?, ?, ?);`,
			tr.id, tr.train, tr.svc, tr.schedArr, tr.delay, tr.cancelled)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func arr(day string, train int, svc string, delay int) seedTrip {
	return seedTrip{id: fmt.Sprintf("%s-%d", day, train), svc: svc, train: train, schedArr: sp("10:00"), delay: ip(delay), cancelled: 0}
}

type statsRow struct {
	Trips, Cancelled, OnTime, Delayed, DelaySum, DelayCount, L10, L1030, L30, Max int
}

func loadStats(t *testing.T, db *sql.DB, day, svc string) statsRow {
	t.Helper()
	var r statsRow
	err := db.QueryRow(`SELECT trips, cancelled, on_time, delayed, delay_sum, delay_count, late_up_to_10, late_10_30, late_over_30, max_delay FROM daily_stats WHERE day = ? AND service_type = ?;`, day, svc).
		Scan(&r.Trips, &r.Cancelled, &r.OnTime, &r.Delayed, &r.DelaySum, &r.DelayCount, &r.L10, &r.L1030, &r.L30, &r.Max)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func dump(t *testing.T, db *sql.DB, query string) [][]any {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var out [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		out = append(out, vals)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRollupThresholdsAndBands(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	seed(t, db,
		arr(day, 1, "U", 3), arr(day, 2, "U", 4), arr(day, 3, "U", 11),
		arr(day, 4, "IC", 5), arr(day, 5, "IC", 6), arr(day, 6, "IC", 10), arr(day, 7, "IC", 11),
		arr(day, 8, "IC", 30), arr(day, 9, "IC", 31), arr(day, 10, "IC", 0),
	)
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}

	u := loadStats(t, db, day, "U")
	if want := (statsRow{Trips: 3, OnTime: 1, Delayed: 2, DelaySum: 18, DelayCount: 3, L10: 1, L1030: 1, Max: 11}); u != want {
		t.Errorf("U = %+v, want %+v", u, want)
	}
	ic := loadStats(t, db, day, "IC")
	if want := (statsRow{Trips: 7, OnTime: 2, Delayed: 5, DelaySum: 93, DelayCount: 7, L10: 2, L1030: 2, L30: 1, Max: 31}); ic != want {
		t.Errorf("IC = %+v, want %+v", ic, want)
	}
	for _, r := range []statsRow{u, ic} {
		if r.OnTime+r.Delayed != r.DelayCount || r.OnTime+r.L10+r.L1030+r.L30 != r.DelayCount {
			t.Errorf("bands do not partition: %+v", r)
		}
	}
}

func TestRollupNullDelayAndDepartureOnly(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	seed(t, db,
		seedTrip{id: day + "-1", svc: "R", train: 1, schedArr: sp("10:00"), delay: nil, cancelled: 0},
		seedTrip{id: day + "-2", svc: "R", train: 2, schedArr: nil, delay: nil, cancelled: nil},
		seedTrip{id: day + "-3", svc: "R", train: 3, schedArr: nil, delay: ip(40), cancelled: 0},
	)
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	got := loadStats(t, db, day, "R")
	if want := (statsRow{Trips: 3, OnTime: 1, DelayCount: 1}); got != want {
		t.Errorf("R = %+v, want %+v", got, want)
	}
}

func TestRollupCancelledExcludedFromDelayMetrics(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	cancelled := arr(day, 1, "IC", 50)
	cancelled.cancelled = 1
	seed(t, db, cancelled, arr(day, 2, "IC", 7))
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	got := loadStats(t, db, day, "IC")
	if want := (statsRow{Trips: 2, Cancelled: 1, Delayed: 1, DelaySum: 7, DelayCount: 1, L10: 1, Max: 7}); got != want {
		t.Errorf("IC = %+v, want %+v", got, want)
	}
	if n := len(dump(t, db, `SELECT 1 FROM daily_worst_trips WHERE trip_id = '`+day+`-1';`)); n != 0 {
		t.Errorf("cancelled trip in worst trips")
	}
}

func TestRollupIdempotent(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	seed(t, db, arr(day, 1, "IC", 9), arr(day, 2, "U", 20), arr(day, 3, "R", 0))
	snapshot := func() [][]any {
		return append(dump(t, db, `SELECT * FROM daily_stats ORDER BY 1, 2;`), dump(t, db, `SELECT * FROM daily_worst_trips ORDER BY 1, 2, 3;`)...)
	}
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	first := snapshot()
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	if second := snapshot(); !reflect.DeepEqual(first, second) || len(first) == 0 {
		t.Errorf("rollup not idempotent:\n%v\n%v", first, second)
	}
}

func TestRollupWorstTripsOrderingTiesLimit(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	var trips []seedTrip
	for i := 1; i <= 12; i++ {
		trips = append(trips, arr(day, 100+i, "IC", i+5))
	}
	trips = append(trips, arr(day, 50, "IC", 17), arr(day, 40, "IC", 17))
	cancelled := arr(day, 1, "IC", 99)
	cancelled.cancelled = 1
	trips = append(trips, cancelled, arr(day, 200, "R", 7))
	seed(t, db, trips...)
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}

	rows := dump(t, db, `SELECT rank, train_number, delay_minutes FROM daily_worst_trips WHERE service_type = 'IC' ORDER BY rank;`)
	want := [][2]int64{{40, 17}, {50, 17}, {112, 17}, {111, 16}, {110, 15}, {109, 14}, {108, 13}, {107, 12}, {106, 11}, {105, 10}}
	if len(rows) != len(want) {
		t.Fatalf("got %d worst rows, want %d: %v", len(rows), len(want), rows)
	}
	for i, r := range rows {
		if r[0].(int64) != int64(i+1) || r[1].(int64) != want[i][0] || r[2].(int64) != want[i][1] {
			t.Errorf("rank %d = %v, want train %d delay %d", i+1, r, want[i][0], want[i][1])
		}
	}
	if n := len(dump(t, db, `SELECT 1 FROM daily_worst_trips WHERE service_type = 'R';`)); n != 1 {
		t.Errorf("R worst rows = %d, want 1", n)
	}
}

func TestRollupCountsMatchTrips(t *testing.T) {
	db := newTestDB(t)
	cancelled := arr("2026-11-05", 3, "R", 0)
	cancelled.cancelled = 1
	seed(t, db,
		arr("2026-11-04", 1, "IC", 2), arr("2026-11-05", 1, "IC", 2), arr("2026-11-05", 2, "U", 8), cancelled,
		seedTrip{id: "2026-11-05-9", svc: "AP", train: 9},
		arr("2026-11-06", 1, "IC", 2),
	)
	if err := rollupRange(db, "2026-11-04", "2026-11-06"); err != nil {
		t.Fatal(err)
	}
	for _, day := range []string{"2026-11-04", "2026-11-05"} {
		var got, cancelledGot, want, cancelledWant int
		if err := db.QueryRow(`SELECT SUM(trips), SUM(cancelled) FROM daily_stats WHERE day = ?;`, day).Scan(&got, &cancelledGot); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(is_cancelled), 0) FROM trips WHERE substr(id, 1, 10) = ?;`, day).Scan(&want, &cancelledWant); err != nil {
			t.Fatal(err)
		}
		if got != want || cancelledGot != cancelledWant {
			t.Errorf("%s: stats %d/%d, trips %d/%d", day, got, cancelledGot, want, cancelledWant)
		}
	}
	if n := len(dump(t, db, `SELECT 1 FROM daily_stats WHERE day = '2026-11-06';`)); n != 0 {
		t.Errorf("day outside range was rolled up")
	}
}

func TestYesterdayToRollup(t *testing.T) {
	if _, ok := yesterdayToRollup(at(7, 3, 59)); ok {
		t.Error("due before 04:00")
	}
	day, ok := yesterdayToRollup(at(7, 4, 0))
	if !ok || day != "2026-11-06" {
		t.Errorf("got %q %v, want 2026-11-06 true", day, ok)
	}
}

func TestRollupYesterdayIfDue(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, arr("2026-11-06", 1, "IC", 9))
	count := func() int { return len(dump(t, db, `SELECT 1 FROM daily_stats;`)) }

	if err := rollupYesterdayIfDue(db, at(7, 3, 0)); err != nil || count() != 0 {
		t.Fatalf("rolled up before 04:00: err=%v rows=%d", err, count())
	}
	if err := rollupYesterdayIfDue(db, at(7, 4, 25)); err != nil || count() != 1 {
		t.Fatalf("not rolled up after 04:00: err=%v rows=%d", err, count())
	}
	// a late-added U trip only shows up if yesterday were recomputed
	seed(t, db, arr("2026-11-06", 2, "U", 9))
	if err := rollupYesterdayIfDue(db, at(7, 10, 0)); err != nil {
		t.Fatal(err)
	}
	if count() != 1 {
		t.Errorf("already rolled-up day was recomputed")
	}
}

func TestBackfillRollups(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		arr("2026-09-30", 1, "IC", 2), arr("2026-10-01", 1, "IC", 8), arr("2026-10-31", 1, "R", 8),
		arr("2026-11-06", 1, "IC", 2), arr("2026-11-07", 1, "IC", 2),
	)
	if err := backfillRollups(db, at(7, 12, 0)); err != nil {
		t.Fatal(err)
	}
	days := dump(t, db, `SELECT day FROM daily_stats ORDER BY day, service_type;`)
	if len(days) != 4 || days[0][0] != "2026-09-30" || days[3][0] != "2026-11-06" {
		t.Errorf("backfilled days = %v, want 09-30, 10-01, 10-31, 11-06 (not today)", days)
	}
}

func TestBackfillRespectsRollupHour(t *testing.T) {
	for _, tc := range []struct {
		now      time.Time
		wantDays []string
	}{
		{at(7, 2, 0), []string{"2026-11-05"}},
		{at(7, 5, 0), []string{"2026-11-05", "2026-11-06"}},
	} {
		db := newTestDB(t)
		seed(t, db, arr("2026-11-05", 1, "IC", 8), arr("2026-11-06", 1, "IC", 8), arr("2026-11-07", 1, "IC", 8))
		if err := backfillRollups(db, tc.now); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, row := range dump(t, db, `SELECT DISTINCT day FROM daily_stats ORDER BY day;`) {
			got = append(got, row[0].(string))
		}
		if !reflect.DeepEqual(got, tc.wantDays) {
			t.Errorf("at %v: backfilled days = %v, want %v", tc.now, got, tc.wantDays)
		}
	}
}

func TestBackfillEmptyTrips(t *testing.T) {
	if err := backfillRollups(newTestDB(t), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestRollupWorstTripsOnlyDelayed(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	seed(t, db,
		arr(day, 1, "IC", 5), arr(day, 2, "IC", 0), arr(day, 3, "IC", -4), arr(day, 4, "IC", 6),
		arr(day, 5, "U", 3), arr(day, 6, "U", 4),
	)
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	rows := dump(t, db, `SELECT service_type, rank, train_number FROM daily_worst_trips ORDER BY service_type, rank;`)
	want := [][]any{{"IC", int64(1), int64(4)}, {"U", int64(1), int64(6)}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("worst = %v, want %v", rows, want)
	}
}

func TestRollupNullServiceType(t *testing.T) {
	db := newTestDB(t)
	const day = "2026-11-05"
	if _, err := db.Exec(`INSERT INTO trips (id, train_number, scheduled_arrival, delay_minutes, is_cancelled) VALUES (?, 1, '10:00', 9, 0);`, day+"-1"); err != nil {
		t.Fatal(err)
	}
	if err := rollupDay(db, day); err != nil {
		t.Fatal(err)
	}
	if got := loadStats(t, db, day, ""); got.Trips != 1 || got.Delayed != 1 {
		t.Errorf("'' = %+v", got)
	}
}

func TestRollupZeroTripDayDoesNothing(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, arr("2026-11-05", 1, "IC", 9))
	if err := rollupDay(db, "2026-11-05"); err != nil {
		t.Fatal(err)
	}
	// existing rows for a day without trips are left alone, proving no DELETE ran
	if err := rollupDay(db, "2026-11-06"); err != nil {
		t.Fatal(err)
	}
	if err := rollupYesterdayIfDue(db, at(7, 5, 0)); err != nil {
		t.Fatal(err)
	}
	if n := len(dump(t, db, `SELECT 1 FROM daily_stats;`)); n != 1 {
		t.Errorf("daily_stats rows = %d, want 1", n)
	}
}
