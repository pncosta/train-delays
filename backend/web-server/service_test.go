package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustParseDay(t *testing.T, day string) time.Time {
	t.Helper()
	parsed, err := time.Parse(dayLayout, day)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// 2026-11-22 12:00 UTC: today is the 22nd, yesterday the 21st.
var testNow = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

type statsReadRecorder struct {
	reads   int
	fromDay string
	toDay   string
}

func (recorder *statsReadRecorder) read(_ context.Context, fromDay, toDay string) ([]DailyStats, error) {
	recorder.reads++
	recorder.fromDay = fromDay
	recorder.toDay = toDay
	return []DailyStats{{Day: fromDay, ServiceType: "AP", Trips: 1}}, nil
}

func TestIsCacheableDay(t *testing.T) {
	tests := []struct {
		day  string
		want bool
	}{
		{"2026-11-20", true},
		{"2026-11-21", false},
		{"2026-11-22", false},
		{"2026-11-23", false},
	}
	for _, test := range tests {
		if got := isCacheableDay(mustParseDay(t, test.day), testNow); got != test.want {
			t.Errorf("isCacheableDay(%s) = %v, want %v", test.day, got, test.want)
		}
	}
}

func TestLoadStatsRepeatedRequestDoesNotRead(t *testing.T) {
	recorder := &statsReadRecorder{}
	cache := newDayCache()
	from := mustParseDay(t, "2026-11-10")
	to := mustParseDay(t, "2026-11-12")

	for range 2 {
		_, err := loadStats(context.Background(), cache, testNow, from, to, recorder.read)
		if err != nil {
			t.Fatal(err)
		}
	}

	if recorder.reads != 1 {
		t.Errorf("got %d reads, want 1; days with no rows must be cached too", recorder.reads)
	}
	if recorder.fromDay != "2026-11-10" || recorder.toDay != "2026-11-12" {
		t.Errorf("read %s..%s, want the whole requested range", recorder.fromDay, recorder.toDay)
	}
}

func TestLoadStatsAlwaysRereadsYesterdayAndToday(t *testing.T) {
	recorder := &statsReadRecorder{}
	cache := newDayCache()
	from := mustParseDay(t, "2026-11-19")
	to := mustParseDay(t, "2026-11-22")

	for range 2 {
		_, err := loadStats(context.Background(), cache, testNow, from, to, recorder.read)
		if err != nil {
			t.Fatal(err)
		}
	}

	if recorder.reads != 2 {
		t.Errorf("got %d reads, want 2", recorder.reads)
	}
}

func TestLoadStatsReadsOnlyFromFirstUncachedDay(t *testing.T) {
	recorder := &statsReadRecorder{}
	cache := newDayCache()
	ctx := context.Background()

	_, err := loadStats(ctx, cache, testNow, mustParseDay(t, "2026-11-10"), mustParseDay(t, "2026-11-12"), recorder.read)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := loadStats(ctx, cache, testNow, mustParseDay(t, "2026-11-10"), mustParseDay(t, "2026-11-15"), recorder.read)
	if err != nil {
		t.Fatal(err)
	}

	if recorder.reads != 2 || recorder.fromDay != "2026-11-13" || recorder.toDay != "2026-11-15" {
		t.Errorf("%d reads, last %s..%s; want a second read of 2026-11-13..2026-11-15", recorder.reads, recorder.fromDay, recorder.toDay)
	}
	if len(rows) != 2 || rows[0].Day != "2026-11-10" || rows[1].Day != "2026-11-13" {
		t.Errorf("want the cached 2026-11-10 row then the fresh 2026-11-13 row, got %+v", rows)
	}
}

func TestLoadStatsReadErrorLeavesCacheEmpty(t *testing.T) {
	cache := newDayCache()
	failingRead := func(context.Context, string, string) ([]DailyStats, error) {
		return nil, errors.New("db down")
	}
	from := mustParseDay(t, "2026-11-10")
	to := mustParseDay(t, "2026-11-12")

	_, err := loadStats(context.Background(), cache, testNow, from, to, failingRead)

	if err == nil {
		t.Fatal("want the read error")
	}
	if len(cache.stats) != 0 {
		t.Errorf("cache has %d days, want 0", len(cache.stats))
	}
}

func TestLoadWorstTripsRepeatedRequestDoesNotRead(t *testing.T) {
	reads := 0
	read := func(_ context.Context, fromDay, toDay string) ([]DailyWorstTrip, error) {
		reads++
		return []DailyWorstTrip{{Day: fromDay, TripID: "a"}}, nil
	}
	cache := newDayCache()
	from := mustParseDay(t, "2026-11-10")
	to := mustParseDay(t, "2026-11-12")

	for range 2 {
		_, err := loadWorstTrips(context.Background(), cache, testNow, from, to, read)
		if err != nil {
			t.Fatal(err)
		}
	}

	if reads != 1 {
		t.Errorf("got %d reads, want 1", reads)
	}
}

func TestTopWorstTrips(t *testing.T) {
	trips := []DailyWorstTrip{
		{TripID: "c", TrainNumber: 300, ServiceType: "AP", DelayMinutes: 20},
		{TripID: "a", TrainNumber: 100, ServiceType: "AP", DelayMinutes: 20},
		{TripID: "d", TrainNumber: 50, ServiceType: "IC", DelayMinutes: 5},
		{TripID: "b", TrainNumber: 200, ServiceType: "IC", DelayMinutes: 90},
		{TripID: "a2", TrainNumber: 100, ServiceType: "AP", DelayMinutes: 20},
	}
	tests := []struct {
		name        string
		serviceType string
		limit       int
		wantIDs     []string
	}{
		{"all types ordered by delay, train number, trip id", "", 4, []string{"b", "a", "a2", "c"}},
		{"filtered by service type", "IC", 10, []string{"b", "d"}},
		{"no match", "U", 10, []string{}},
	}
	for _, test := range tests {
		got := topWorstTrips(trips, test.serviceType, test.limit)
		if len(got) != len(test.wantIDs) {
			t.Errorf("%s: got %d trips, want %d", test.name, len(got), len(test.wantIDs))
			continue
		}
		for i, wantID := range test.wantIDs {
			if got[i].TripID != wantID {
				t.Errorf("%s: position %d = %s, want %s", test.name, i, got[i].TripID, wantID)
			}
		}
	}
}

func intPointer(value int) *int { return &value }

func findStats(response *DashboardResponse, serviceType string) ServiceStats {
	for _, stats := range response.Breakdown {
		if stats.ServiceType == serviceType {
			return stats
		}
	}
	return ServiceStats{}
}

func TestSummarizeGroupsThresholds(t *testing.T) {
	groups := []summaryGroup{
		{ServiceType: "U", DelayMinutes: intPointer(4), HasScheduledArrival: true, Count: 1},
		{ServiceType: "U", DelayMinutes: intPointer(3), HasScheduledArrival: true, Count: 2},
		{ServiceType: "IC", DelayMinutes: intPointer(4), HasScheduledArrival: true, Count: 1},
		{ServiceType: "IC", DelayMinutes: intPointer(5), HasScheduledArrival: true, Count: 1},
		{ServiceType: "IC", DelayMinutes: intPointer(6), HasScheduledArrival: true, Count: 1},
	}

	response := summarizeGroups(groups)

	urban := findStats(response, "U")
	if urban.DelayedCount != 1 || urban.OnTimeCount != 2 {
		t.Errorf("U: %+v, want 1 delayed and 2 on time", urban)
	}
	intercity := findStats(response, "IC")
	if intercity.DelayedCount != 1 || intercity.OnTimeCount != 2 {
		t.Errorf("IC: %+v, want 1 delayed and 2 on time", intercity)
	}
	system := findStats(response, totalSystemServiceType)
	if system.TotalTrips != 6 || system.DelayedCount != 2 || system.OnTimeCount != 4 {
		t.Errorf("TOTAL_SYSTEM: %+v", system)
	}
	if response.Breakdown[len(response.Breakdown)-1].ServiceType != totalSystemServiceType {
		t.Error("TOTAL_SYSTEM should be last")
	}
}

func TestSummarizeGroupsNullDelayAndCancelled(t *testing.T) {
	groups := []summaryGroup{
		{ServiceType: "AP", DelayMinutes: nil, HasScheduledArrival: true, Count: 3},
		{ServiceType: "AP", DelayMinutes: intPointer(40), HasScheduledArrival: false, Count: 10},
		{ServiceType: "AP", DelayMinutes: nil, IsCancelled: true, HasScheduledArrival: false, Count: 1},
		{ServiceType: "AP", DelayMinutes: intPointer(8), IsCancelled: true, HasScheduledArrival: true, Count: 2},
	}

	apStats := findStats(summarizeGroups(groups), "AP")

	if apStats.TotalTrips != 16 || apStats.CancelledCount != 3 || apStats.OnTimeCount != 3 || apStats.DelayedCount != 0 {
		t.Errorf("AP: %+v, want 16 total, 3 cancelled, 3 on time (a delay without an arrival isn't eligible)", apStats)
	}
	if apStats.AvgDelay != 0 {
		t.Errorf("AP avg delay = %v, want 0 (only the 3 eligible trips, all with NULL delay, count)", apStats.AvgDelay)
	}
}
