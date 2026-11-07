package main

import (
	"database/sql"
	"testing"
	"time"
	"train-delays/shared"

	_ "modernc.org/sqlite"
)

var testLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		panic(err)
	}
	return loc
}()

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// modernc SQLite shares libsql's dialect, but libsql's driver param conversion isn't covered
	// each connection to :memory: is its own database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if err := applySchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSchemaCreatesSummaryIndex(t *testing.T) {
	db := newTestDB(t)
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = 'idx_trips_optimized_summary'`).Scan(&name)
	if err != nil {
		t.Fatalf("summary index missing: %v", err)
	}
}

func at(day, hour, min int) time.Time {
	return time.Date(2026, 11, day, hour, min, 0, 0, testLoc)
}

func sp(s string) *string { return &s }
func ip(i int) *int       { return &i }

func testTrip(train int, dep, eta, arr, etd *string, delay *int) Trip {
	return Trip{
		TrainNumber:      train,
		TrainService:     TrainServiceInfo{Code: "IC"},
		TrainOrigin:      shared.StationInfo{Code: "A"},
		TrainDestination: shared.StationInfo{Code: "B"},
		DepartureTime:    dep,
		ETD:              etd,
		ArrivalTime:      arr,
		ETA:              eta,
		Delay:            delay,
	}
}

type tripRow struct {
	schedDep, schedArr, actualDep, actualArr sql.NullString
	delay                                    sql.NullInt64
}

func loadRows(t *testing.T, db *sql.DB) map[string]tripRow {
	t.Helper()
	rows, err := db.Query(`SELECT id, scheduled_departure, scheduled_arrival, actual_departure, actual_arrival, delay_minutes FROM trips;`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]tripRow{}
	for rows.Next() {
		var id string
		var r tripRow
		if err := rows.Scan(&id, &r.schedDep, &r.schedArr, &r.actualDep, &r.actualArr, &r.delay); err != nil {
			t.Fatal(err)
		}
		out[id] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func mustStart(t *testing.T, db *sql.DB, now time.Time, trip Trip) {
	t.Helper()
	if err := InsertStartingTrip(db, now, trip); err != nil {
		t.Fatal(err)
	}
}

func mustEnd(t *testing.T, db *sql.DB, now time.Time, trip Trip) {
	t.Helper()
	if err := InsertEndingTrip(db, now, trip); err != nil {
		t.Fatal(err)
	}
}

func wantStr(t *testing.T, field string, got sql.NullString, want string) {
	t.Helper()
	if !got.Valid || got.String != want {
		t.Errorf("%s = %v, want %q", field, got, want)
	}
}

func wantRow(t *testing.T, rows map[string]tripRow, id string) tripRow {
	t.Helper()
	r, ok := rows[id]
	if !ok {
		t.Fatalf("row %s missing; rows: %v", id, rows)
	}
	return r
}

func TestStartingTripKeepsActualDepartureAcrossMidnight(t *testing.T) {
	db := newTestDB(t)
	trip := testTrip(18904, sp("23:50"), nil, sp("02:00"), sp("23:50"), nil)
	mustStart(t, db, at(5, 23, 50), trip)

	trip.ETD = nil
	mustStart(t, db, at(6, 0, 0), trip)

	rows := loadRows(t, db)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %v", len(rows), rows)
	}
	r := wantRow(t, rows, "2026-11-05-18904")
	wantStr(t, "actual_departure", r.actualDep, "23:50")
	wantStr(t, "scheduled_departure", r.schedDep, "23:50")
}

func TestEndingTripRowId(t *testing.T) {
	type run struct {
		now time.Time
		arr string
		eta string
	}
	tests := []struct {
		name    string
		seedDep string
		seedNow time.Time
		// zero means no second starting run
		seedNextDayNow time.Time
		runs           []run
		wantIDs        []string
		wantRow        string
		wantArr        string
	}{
		{
			name:    "post-midnight ETA updates land on the day-D row",
			seedDep: "22:45",
			seedNow: at(5, 22, 45),
			runs: []run{
				{at(6, 1, 0), "00:42", "01:09"},
				{at(6, 1, 25), "00:42", "01:10"},
			},
			wantIDs: []string{"2026-11-05-4437"},
			wantRow: "2026-11-05-4437",
			wantArr: "01:10",
		},
		{
			name:    "late-evening ETA past midnight seen before midnight lands on day D",
			seedDep: "23:30",
			seedNow: at(5, 23, 30),
			runs: []run{
				{at(5, 23, 50), "00:05", "00:05"},
			},
			wantIDs: []string{"2026-11-05-4437"},
			wantRow: "2026-11-05-4437",
			wantArr: "00:05",
		},
		{
			name:           "own-day early train arrival merges into its own D+1 row, not the previous day's",
			seedDep:        "00:30",
			seedNow:        at(5, 0, 30),
			seedNextDayNow: at(6, 0, 30),
			runs: []run{
				{at(6, 1, 10), "01:10", "01:10"},
			},
			wantIDs: []string{"2026-11-05-4437", "2026-11-06-4437"},
			wantRow: "2026-11-06-4437",
			wantArr: "01:10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			mustStart(t, db, tt.seedNow, testTrip(4437, sp(tt.seedDep), nil, nil, nil, nil))
			if !tt.seedNextDayNow.IsZero() {
				mustStart(t, db, tt.seedNextDayNow, testTrip(4437, sp(tt.seedDep), nil, nil, nil, nil))
			}
			for _, r := range tt.runs {
				mustEnd(t, db, r.now, testTrip(4437, sp(tt.seedDep), sp(r.eta), sp(r.arr), nil, nil))
			}

			rows := loadRows(t, db)
			if len(rows) != len(tt.wantIDs) {
				t.Fatalf("got %d rows, want %v: %v", len(rows), tt.wantIDs, rows)
			}
			for _, id := range tt.wantIDs {
				wantRow(t, rows, id)
			}
			wantStr(t, "actual_arrival", rows[tt.wantRow].actualArr, tt.wantArr)
			if tt.wantRow != tt.wantIDs[0] && rows[tt.wantIDs[0]].actualArr.Valid {
				t.Errorf("%s actual_arrival = %v, want untouched", tt.wantIDs[0], rows[tt.wantIDs[0]].actualArr)
			}
		})
	}
}

func TestEndingTripDelay(t *testing.T) {
	tests := []struct {
		name   string
		delays []*int
		want   int64
	}{
		{"nil delay after a known delay keeps it", []*int{ip(13), nil}, 13},
		{"fresh insert with nil delay stores 0", []*int{nil}, 0},
		{"explicit 0 overwrites a previous delay", []*int{ip(13), ip(0)}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			for i, d := range tt.delays {
				now := at(5, 10, 0).Add(time.Duration(i) * 25 * time.Minute)
				mustEnd(t, db, now, testTrip(100, nil, sp("10:00"), sp("10:00"), nil, d))
			}
			r := wantRow(t, loadRows(t, db), "2026-11-05-100")
			if !r.delay.Valid || r.delay.Int64 != tt.want {
				t.Errorf("delay_minutes = %v, want %d", r.delay, tt.want)
			}
		})
	}
}
