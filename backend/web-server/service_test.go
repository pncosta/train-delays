package main

import (
	"context"
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

func TestPeriodLabel(t *testing.T) {
	tests := []struct {
		day         string
		granularity string
		want        string
	}{
		{"2026-11-18", granularityDay, "2026-11-18"},
		{"2026-11-18", granularityWeek, "2026-11-16"},
		{"2026-11-16", granularityWeek, "2026-11-16"},
		{"2026-11-22", granularityWeek, "2026-11-16"},
		{"2026-01-01", granularityWeek, "2025-12-29"},
		{"2025-12-31", granularityWeek, "2025-12-29"},
		{"2026-11-18", granularityMonth, "2026-11-01"},
		{"2026-02-28", granularityMonth, "2026-02-01"},
	}
	for _, test := range tests {
		got := periodLabel(mustParseDay(t, test.day), test.granularity)
		if got != test.want {
			t.Errorf("periodLabel(%s, %s) = %s, want %s", test.day, test.granularity, got, test.want)
		}
	}
}

func TestCountPeriods(t *testing.T) {
	from := mustParseDay(t, "2025-12-29")
	to := mustParseDay(t, "2026-01-11")
	tests := []struct {
		granularity string
		want        int
	}{
		{granularityDay, 14},
		{granularityWeek, 2},
		{granularityMonth, 2},
	}
	for _, test := range tests {
		if got := countPeriods(from, to, test.granularity); got != test.want {
			t.Errorf("%s periods = %d, want %d", test.granularity, got, test.want)
		}
	}
}

// 2026-11-22 12:00 Lisbon: today is the 22nd, yesterday the 21st.
var testNow = time.Date(2026, 11, 22, 12, 0, 0, 0, lisbonLocation)

type readCounter struct {
	reads    int
	fromDay  string
	toDay    string
	statsRow map[string][]DailyStats
}

func (counter *readCounter) read(_ context.Context, fromDay, toDay string) ([]DailyStats, error) {
	counter.reads++
	counter.fromDay = fromDay
	counter.toDay = toDay
	var rows []DailyStats
	for _, day := range daysBetweenStrings(fromDay, toDay) {
		rows = append(rows, counter.statsRow[day]...)
	}
	return rows, nil
}

func daysBetweenStrings(fromDay, toDay string) []string {
	from, _ := time.Parse(dayLayout, fromDay)
	to, _ := time.Parse(dayLayout, toDay)
	return daysBetween(from, to)
}

func TestLoadStatsCaching(t *testing.T) {
	counter := &readCounter{statsRow: map[string][]DailyStats{
		"2026-11-18": {{Day: "2026-11-18", ServiceType: "AP", Trips: 1}},
		"2026-11-20": {{Day: "2026-11-20", ServiceType: "AP", Trips: 2}},
	}}
	cache := newDayCache()
	ctx := context.Background()
	from := mustParseDay(t, "2026-11-17")
	to := mustParseDay(t, "2026-11-22")

	rows, err := loadStats(ctx, cache, testNow, from, to, counter.read)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows, want 2", len(rows))
	}
	if counter.reads != 1 || counter.fromDay != "2026-11-17" || counter.toDay != "2026-11-21" {
		t.Errorf("first request: %d reads of %s..%s, want 1 read of 2026-11-17..2026-11-21", counter.reads, counter.fromDay, counter.toDay)
	}

	// yesterday had no rollup rows, so it is read again; the other days are cached
	_, err = loadStats(ctx, cache, testNow, from, to, counter.read)
	if err != nil {
		t.Fatal(err)
	}
	if counter.reads != 2 || counter.fromDay != "2026-11-21" || counter.toDay != "2026-11-21" {
		t.Errorf("second request: %d reads of %s..%s, want a second read of only 2026-11-21", counter.reads, counter.fromDay, counter.toDay)
	}

	counter.statsRow["2026-11-21"] = []DailyStats{{Day: "2026-11-21", ServiceType: "AP", Trips: 3}}
	rows, err = loadStats(ctx, cache, testNow, from, to, counter.read)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || counter.reads != 3 {
		t.Errorf("after rollup: %d rows and %d reads, want 3 rows and 3 reads", len(rows), counter.reads)
	}

	_, err = loadStats(ctx, cache, testNow, from, to, counter.read)
	if err != nil {
		t.Fatal(err)
	}
	if counter.reads != 3 {
		t.Errorf("repeated request read the DB: %d reads, want 3", counter.reads)
	}
}

func TestLoadStatsNeverReadsToday(t *testing.T) {
	counter := &readCounter{}
	today := mustParseDay(t, "2026-11-22")

	_, err := loadStats(context.Background(), newDayCache(), testNow, today, today, counter.read)
	if err != nil {
		t.Fatal(err)
	}
	if counter.reads != 0 {
		t.Errorf("read %d times for a today-only range, want 0", counter.reads)
	}
}

func TestBuildTimeseriesWeekAcrossYearBoundaryWithTotal(t *testing.T) {
	rows := []DailyStats{
		{Day: "2025-12-31", ServiceType: "AP", Trips: 10, Cancelled: 1, OnTime: 6, Delayed: 3, DelaySum: 30, DelayCount: 9, LateUpTo10: 2, Late10To30: 1, MaxDelay: 12},
		{Day: "2026-01-01", ServiceType: "AP", Trips: 20, Cancelled: 2, OnTime: 10, Delayed: 8, DelaySum: 90, DelayCount: 18, LateUpTo10: 4, Late10To30: 3, LateOver30: 1, MaxDelay: 45},
		{Day: "2026-01-01", ServiceType: "IC", Trips: 5, OnTime: 5, DelayCount: 5, MaxDelay: 2},
		{Day: "2026-01-05", ServiceType: "AP", Trips: 7, OnTime: 7, DelayCount: 7},
	}

	periods, err := buildTimeseries(rows, granularityWeek)
	if err != nil {
		t.Fatal(err)
	}
	if len(periods) != 2 {
		t.Fatalf("got %d periods, want 2", len(periods))
	}

	firstWeek := periods[0]
	if firstWeek.Period != "2025-12-29" {
		t.Errorf("first period = %s, want 2025-12-29", firstWeek.Period)
	}
	if len(firstWeek.Stats) != 3 {
		t.Fatalf("got %d stats, want AP, IC, TOTAL", len(firstWeek.Stats))
	}
	if firstWeek.Stats[0].ServiceType != "AP" || firstWeek.Stats[1].ServiceType != "IC" || firstWeek.Stats[2].ServiceType != totalServiceType {
		t.Errorf("unexpected order: %+v", firstWeek.Stats)
	}

	apStats := firstWeek.Stats[0]
	if apStats.Trips != 30 || apStats.Cancelled != 3 || apStats.OnTime != 16 || apStats.Delayed != 11 ||
		apStats.LateUpTo10 != 6 || apStats.Late10To30 != 4 || apStats.LateOver30 != 1 || apStats.MaxDelay != 45 {
		t.Errorf("AP week sums wrong: %+v", apStats)
	}
	if apStats.AvgDelay != 120.0/27.0 {
		t.Errorf("AP avg delay = %v, want %v", apStats.AvgDelay, 120.0/27.0)
	}

	total := firstWeek.Stats[2]
	if total.Trips != 35 || total.OnTime != 21 || total.Delayed != 11 || total.Cancelled != 3 || total.MaxDelay != 45 {
		t.Errorf("TOTAL sums wrong: %+v", total)
	}
	if total.AvgDelay != 120.0/32.0 {
		t.Errorf("TOTAL avg delay = %v, want %v", total.AvgDelay, 120.0/32.0)
	}

	if periods[1].Period != "2026-01-05" {
		t.Errorf("second period = %s, want 2026-01-05", periods[1].Period)
	}
}

func TestBuildTimeseriesMonthSumsDays(t *testing.T) {
	rows := []DailyStats{
		{Day: "2026-10-31", ServiceType: "U", Trips: 3, Delayed: 1},
		{Day: "2026-11-01", ServiceType: "U", Trips: 4, Delayed: 2},
		{Day: "2026-11-30", ServiceType: "U", Trips: 5, Delayed: 3},
	}

	periods, err := buildTimeseries(rows, granularityMonth)
	if err != nil {
		t.Fatal(err)
	}
	if len(periods) != 2 || periods[0].Period != "2026-10-01" || periods[1].Period != "2026-11-01" {
		t.Fatalf("unexpected periods: %+v", periods)
	}
	november := periods[1].Stats[0]
	if november.Trips != 9 || november.Delayed != 5 {
		t.Errorf("november sums wrong: %+v", november)
	}
}

func TestBuildTimeseriesNoRows(t *testing.T) {
	periods, err := buildTimeseries(nil, granularityDay)
	if err != nil {
		t.Fatal(err)
	}
	if periods == nil || len(periods) != 0 {
		t.Errorf("want empty non-nil periods, got %#v", periods)
	}
}

func TestTopWorstTrips(t *testing.T) {
	trips := []DailyWorstTrip{
		{TripID: "c", TrainNumber: 300, DelayMinutes: 20},
		{TripID: "a", TrainNumber: 100, DelayMinutes: 20},
		{TripID: "d", TrainNumber: 50, DelayMinutes: 5},
		{TripID: "b", TrainNumber: 200, DelayMinutes: 90},
		{TripID: "a2", TrainNumber: 100, DelayMinutes: 20},
	}

	got := topWorstTrips(trips, 4)

	wantIDs := []string{"b", "a", "a2", "c"}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d trips, want %d", len(got), len(wantIDs))
	}
	for i, wantID := range wantIDs {
		if got[i].TripID != wantID {
			t.Errorf("position %d = %s, want %s", i, got[i].TripID, wantID)
		}
	}
	if trips[0].TripID != "c" {
		t.Error("input slice was reordered")
	}
}

func TestIsFinalDay(t *testing.T) {
	const yesterday = "2026-11-21"
	tests := []struct {
		name     string
		day      string
		hasStats bool
		want     bool
	}{
		{"before yesterday, empty", "2026-11-20", false, true},
		{"before yesterday, with rows", "2026-11-20", true, true},
		{"yesterday with rollup rows", "2026-11-21", true, true},
		{"yesterday without rollup rows", "2026-11-21", false, false},
		{"today with rows", "2026-11-22", true, false},
		{"today empty", "2026-11-22", false, false},
	}
	for _, test := range tests {
		if got := isFinalDay(test.day, test.hasStats, yesterday); got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
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
		{ServiceType: "U", DelayMinutes: intPointer(4), ArrivalSeen: true, Count: 1},
		{ServiceType: "U", DelayMinutes: intPointer(3), ArrivalSeen: true, Count: 2},
		{ServiceType: "IC", DelayMinutes: intPointer(4), ArrivalSeen: true, Count: 1},
		{ServiceType: "IC", DelayMinutes: intPointer(5), ArrivalSeen: true, Count: 1},
		{ServiceType: "IC", DelayMinutes: intPointer(6), ArrivalSeen: true, Count: 1},
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
	system := findStats(response, "TOTAL_SYSTEM")
	if system.TotalTrips != 6 || system.DelayedCount != 2 || system.OnTimeCount != 4 {
		t.Errorf("TOTAL_SYSTEM: %+v", system)
	}
	if response.Breakdown[len(response.Breakdown)-1].ServiceType != "TOTAL_SYSTEM" {
		t.Error("TOTAL_SYSTEM should be last")
	}
}

func TestSummarizeGroupsNullDelayAndCancelled(t *testing.T) {
	groups := []summaryGroup{
		{ServiceType: "AP", DelayMinutes: nil, ArrivalSeen: true, Count: 3},
		{ServiceType: "AP", DelayMinutes: nil, ArrivalSeen: false, Count: 10},
		{ServiceType: "AP", DelayMinutes: intPointer(8), IsCancelled: true, ArrivalSeen: true, Count: 2},
	}

	apStats := findStats(summarizeGroups(groups), "AP")

	if apStats.TotalTrips != 5 || apStats.OnTimeCount != 3 || apStats.CancelledCount != 2 || apStats.DelayedCount != 0 {
		t.Errorf("AP: %+v", apStats)
	}
}
