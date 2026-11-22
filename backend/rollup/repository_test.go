package main

import "testing"

func TestReplaceRollupsReplacesOnlyRange(t *testing.T) {
	db := newTestDB(t)
	old := func(day string) dailyStats { return dailyStats{dayService: dayService{day, "IC"}, trips: 1} }
	if err := replaceRollups(t.Context(), db, "2026-11-01", "2026-11-10", []dailyStats{old("2026-11-05"), old("2026-11-06")}, nil); err != nil {
		t.Fatal(err)
	}
	fresh := dailyStats{dayService: dayService{"2026-11-05", "IC"}, trips: 7}
	if err := replaceRollups(t.Context(), db, "2026-11-05", "2026-11-06", []dailyStats{fresh}, nil); err != nil {
		t.Fatal(err)
	}
	if got := loadStats(t, db, "2026-11-05", "IC"); got.Trips != 7 {
		t.Errorf("range day not replaced: %+v", got)
	}
	if got := loadStats(t, db, "2026-11-06", "IC"); got.Trips != 1 {
		t.Errorf("day outside range touched: %+v", got)
	}
}
