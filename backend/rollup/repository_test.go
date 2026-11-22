package main

import "testing"

func TestInsertRollupsKeepsExistingRows(t *testing.T) {
	db := newTestDB(t)
	existing := dailyStats{day: "2026-11-05", serviceType: "IC", trips: 1}
	if err := insertRollups(t.Context(), db, []dailyStats{existing}, nil); err != nil {
		t.Fatal(err)
	}
	recomputed := dailyStats{day: "2026-11-05", serviceType: "IC", trips: 7}
	missing := dailyStats{day: "2026-11-06", serviceType: "IC", trips: 3}
	if err := insertRollups(t.Context(), db, []dailyStats{recomputed, missing}, nil); err != nil {
		t.Fatal(err)
	}
	if got := loadStats(t, db, "2026-11-05", "IC"); got.Trips != 1 {
		t.Errorf("existing day overwritten: %+v", got)
	}
	if got := loadStats(t, db, "2026-11-06", "IC"); got.Trips != 3 {
		t.Errorf("missing day not inserted: %+v", got)
	}
}
