package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

func eligibleTrip(id, svc string, train *int64, delay int) tripRow {
	t := tripRow{id: id, serviceType: sql.NullString{String: svc, Valid: true},
		scheduledArrival: sql.NullString{String: "10:00", Valid: true}, delayMinutes: sql.NullInt64{Int64: int64(delay), Valid: true}}
	if train != nil {
		t.trainNumber = sql.NullInt64{Int64: *train, Valid: true}
	}
	return t
}

func TestBuildDailyStatsBoundaries(t *testing.T) {
	tests := []struct {
		svc    string
		delays []int
		want   dailyStats
	}{
		{"U", []int{3, 4, 10, 11, 30, 31}, dailyStats{trips: 6, onTime: 1, delayed: 5, delaySum: 89, delayCount: 6, lateUpTo10: 2, late10To30: 2, lateOver30: 1, maxDelay: 31}},
		{"IC", []int{5, 6, 10, 11, 30, 31}, dailyStats{trips: 6, onTime: 1, delayed: 5, delaySum: 93, delayCount: 6, lateUpTo10: 2, late10To30: 2, lateOver30: 1, maxDelay: 31}},
		{"IC", []int{-4, -2}, dailyStats{trips: 2, onTime: 2, delaySum: -6, delayCount: 2, maxDelay: -2}},
	}
	for _, tc := range tests {
		var trips []tripRow
		for i, d := range tc.delays {
			trips = append(trips, eligibleTrip(fmt.Sprintf("2026-01-01-%d", i), tc.svc, nil, d))
		}
		tc.want.dayService = dayService{"2026-01-01", tc.svc}
		got := buildDailyStats(trips)
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s %v: got %+v, want %+v", tc.svc, tc.delays, got, tc.want)
		}
	}
}

func TestBuildDailyStatsNoEligibleRows(t *testing.T) {
	cancelled := tripRow{id: "2026-01-01-1", isCancelled: sql.NullBool{Bool: true, Valid: true}}
	got := buildDailyStats([]tripRow{cancelled})
	want := []dailyStats{{dayService: dayService{"2026-01-01", ""}, trips: 1, cancelled: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBuildWorstTripsOrderAndCap(t *testing.T) {
	n := func(v int64) *int64 { return &v }
	trips := []tripRow{
		eligibleTrip("2026-01-01-b", "IC", n(2), 20),
		eligibleTrip("2026-01-01-a", "IC", n(2), 20),
		eligibleTrip("2026-01-01-c", "IC", n(1), 20),
		eligibleTrip("2026-01-01-d", "IC", nil, 20),
		eligibleTrip("2026-01-01-e", "IC", n(9), 40),
		eligibleTrip("2026-01-01-f", "IC", n(9), 5),
	}
	var ids []string
	for _, w := range buildWorstTrips(trips) {
		ids = append(ids, w.trip.id[len(w.trip.id)-1:])
	}
	if want := []string{"e", "d", "c", "a", "b"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("order: got %v, want %v", ids, want)
	}

	var many []tripRow
	for i := 0; i < 15; i++ {
		many = append(many, eligibleTrip(fmt.Sprintf("2026-01-01-%02d", i), "IC", n(int64(i)), 10+i))
	}
	got := buildWorstTrips(many)
	if len(got) != worstTripsPerGroup || got[0].rank != 1 || got[0].trip.delay() != 24 {
		t.Errorf("cap: got %d rows, first %+v", len(got), got[0])
	}
}
