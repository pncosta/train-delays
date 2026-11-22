package main

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"
	"train-delays/shared"
)

const (
	dayLayout = "2006-01-02"

	granularityDay   = "day"
	granularityWeek  = "week"
	granularityMonth = "month"

	totalServiceType       = "TOTAL"
	totalSystemServiceType = "TOTAL_SYSTEM"

	urbanServiceType      = "U"
	urbanDelayedMinutes   = 3
	defaultDelayedMinutes = 5

	// keeps a chart's payload small
	maxTimeseriesPoints = 120
	// a month is at most 31 days, so a longer span can't fit in maxTimeseriesPoints at any granularity
	maxTimeseriesDays = maxTimeseriesPoints * 31
	// ten years, so a request can't ask for an unbounded number of rows
	maxWorstTripsRangeDays = 3660

	worstTripsLimit = 10
)

var lisbonLocation = mustLoadLisbon()

func mustLoadLisbon() *time.Location {
	location, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		panic(fmt.Sprintf("failed to load Europe/Lisbon: %v", err))
	}
	return location
}

// Delayed means strictly above the threshold.
func delayedThreshold(serviceType string) int {
	if serviceType == urbanServiceType {
		return urbanDelayedMinutes
	}
	return defaultDelayedMinutes
}

type summaryTotals struct {
	stats      ServiceStats
	delaySum   int
	delayCount int
}

func (totals *summaryTotals) add(group summaryGroup) {
	totals.stats.TotalTrips += group.Count
	if group.IsCancelled {
		totals.stats.CancelledCount += group.Count
		return
	}

	delay := 0
	if group.DelayMinutes != nil {
		delay = *group.DelayMinutes
	}
	totals.delaySum += delay * group.Count
	totals.delayCount += group.Count

	if delay > delayedThreshold(group.ServiceType) {
		totals.stats.DelayedCount += group.Count
		return
	}
	totals.stats.OnTimeCount += group.Count
}

func (totals *summaryTotals) toServiceStats() ServiceStats {
	stats := totals.stats
	if totals.delayCount > 0 {
		stats.AvgDelay = float64(totals.delaySum) / float64(totals.delayCount)
	}
	return stats
}

// A NULL delay counts as 0 only where the arrival was seen; other NULL-delay trips are left out.
func summarizeGroups(groups []summaryGroup) *DashboardResponse {
	totalsByType := map[string]*summaryTotals{}
	systemTotals := &summaryTotals{stats: ServiceStats{ServiceType: totalSystemServiceType}}

	for _, group := range groups {
		if group.DelayMinutes == nil && !group.ArrivalSeen {
			continue
		}
		totals := totalsByType[group.ServiceType]
		if totals == nil {
			totals = &summaryTotals{stats: ServiceStats{ServiceType: group.ServiceType}}
			totalsByType[group.ServiceType] = totals
		}
		totals.add(group)
		systemTotals.add(group)
	}

	response := &DashboardResponse{Breakdown: []ServiceStats{}}
	for _, serviceType := range slices.Sorted(maps.Keys(totalsByType)) {
		response.Breakdown = append(response.Breakdown, totalsByType[serviceType].toServiceStats())
	}
	response.Breakdown = append(response.Breakdown, systemTotals.toServiceStats())
	return response
}

// dayCache holds final days only, so a cached day is never read again.
type dayCache struct {
	mutex      sync.Mutex
	stats      map[string][]DailyStats
	worstTrips map[string][]DailyWorstTrip
}

func newDayCache() *dayCache {
	return &dayCache{
		stats:      map[string][]DailyStats{},
		worstTrips: map[string][]DailyWorstTrip{},
	}
}

// Yesterday is final once the rollup wrote its rows; older days are final even when empty.
func isFinalDay(day string, rollupWroteRows bool, yesterday string) bool {
	if day < yesterday {
		return true
	}
	return day == yesterday && rollupWroteRows
}

func daysBetween(from, to time.Time) []string {
	var days []string
	day := from
	for !day.After(to) {
		days = append(days, day.Format(dayLayout))
		day = day.AddDate(0, 0, 1)
	}
	return days
}

func rangeDays(from, to time.Time) int {
	return int(to.Sub(from).Hours()/24) + 1
}

// finishedDays returns the days of [from, to] before today (Lisbon) and yesterday's date;
// the rollup only writes finished days, so today is never read.
func finishedDays(now time.Time, from, to time.Time) ([]string, string) {
	year, month, dayOfMonth := now.In(lisbonLocation).Date()
	today := time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	lastDay := to
	if lastDay.After(yesterday) {
		lastDay = yesterday
	}
	return daysBetween(from, lastDay), yesterday.Format(dayLayout)
}

type statsReader func(ctx context.Context, fromDay, toDay string) ([]DailyStats, error)

type worstTripsReader func(ctx context.Context, fromDay, toDay string) ([]DailyWorstTrip, error)

// loadStats returns the daily_stats rows of the finished days in [from, to].
// If any of them isn't cached it reads the whole span from the first to the last uncached day in one query.
// The mutex is held during the read so concurrent requests for the same days read once.
func loadStats(ctx context.Context, cache *dayCache, now time.Time, from, to time.Time, readStats statsReader) ([]DailyStats, error) {
	days, yesterday := finishedDays(now, from, to)

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	firstUncached := -1
	lastUncached := -1
	for i, day := range days {
		if _, cached := cache.stats[day]; cached {
			continue
		}
		if firstUncached == -1 {
			firstUncached = i
		}
		lastUncached = i
	}

	freshByDay := map[string][]DailyStats{}
	if firstUncached != -1 {
		fresh, err := readStats(ctx, days[firstUncached], days[lastUncached])
		if err != nil {
			return nil, err
		}
		for _, row := range fresh {
			freshByDay[row.Day] = append(freshByDay[row.Day], row)
		}
		for _, day := range days[firstUncached : lastUncached+1] {
			if isFinalDay(day, len(freshByDay[day]) > 0, yesterday) {
				cache.stats[day] = freshByDay[day]
			}
		}
	}

	var result []DailyStats
	for _, day := range days {
		rows, cached := cache.stats[day]
		if !cached {
			rows = freshByDay[day]
		}
		result = append(result, rows...)
	}
	return result, nil
}

// loadWorstTrips is loadStats for daily_worst_trips.
func loadWorstTrips(ctx context.Context, cache *dayCache, now time.Time, from, to time.Time, readWorstTrips worstTripsReader) ([]DailyWorstTrip, error) {
	days, yesterday := finishedDays(now, from, to)

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	firstUncached := -1
	lastUncached := -1
	for i, day := range days {
		if _, cached := cache.worstTrips[day]; cached {
			continue
		}
		if firstUncached == -1 {
			firstUncached = i
		}
		lastUncached = i
	}

	freshByDay := map[string][]DailyWorstTrip{}
	if firstUncached != -1 {
		fresh, err := readWorstTrips(ctx, days[firstUncached], days[lastUncached])
		if err != nil {
			return nil, err
		}
		for _, trip := range fresh {
			freshByDay[trip.Day] = append(freshByDay[trip.Day], trip)
		}
		for _, day := range days[firstUncached : lastUncached+1] {
			if isFinalDay(day, len(freshByDay[day]) > 0, yesterday) {
				cache.worstTrips[day] = freshByDay[day]
			}
		}
	}

	var result []DailyWorstTrip
	for _, day := range days {
		trips, cached := cache.worstTrips[day]
		if !cached {
			trips = freshByDay[day]
		}
		result = append(result, trips...)
	}
	return result, nil
}

// periodLabel is the day itself, the Monday of its ISO week, or the first of its month.
func periodLabel(day time.Time, granularity string) string {
	switch granularity {
	case granularityWeek:
		daysSinceMonday := (int(day.Weekday()) + 6) % 7
		return day.AddDate(0, 0, -daysSinceMonday).Format(dayLayout)
	case granularityMonth:
		return day.Format("2006-01") + "-01"
	default:
		return day.Format(dayLayout)
	}
}

func countPeriods(from, to time.Time, granularity string) int {
	count := 0
	lastLabel := ""
	day := from
	for !day.After(to) {
		label := periodLabel(day, granularity)
		if label != lastLabel {
			count++
			lastLabel = label
		}
		day = day.AddDate(0, 0, 1)
	}
	return count
}

func addDailyStats(totals *DailyStats, row DailyStats) {
	totals.Trips += row.Trips
	totals.Cancelled += row.Cancelled
	totals.OnTime += row.OnTime
	totals.Delayed += row.Delayed
	totals.DelaySum += row.DelaySum
	totals.DelayCount += row.DelayCount
	totals.LateUpTo10 += row.LateUpTo10
	totals.Late10To30 += row.Late10To30
	totals.LateOver30 += row.LateOver30
	totals.MaxDelay = max(totals.MaxDelay, row.MaxDelay)
}

func toTimeseriesStats(totals DailyStats) TimeseriesStats {
	avgDelay := 0.0
	if totals.DelayCount > 0 {
		avgDelay = float64(totals.DelaySum) / float64(totals.DelayCount)
	}
	return TimeseriesStats{
		ServiceType: totals.ServiceType,
		Trips:       totals.Trips,
		Cancelled:   totals.Cancelled,
		OnTime:      totals.OnTime,
		Delayed:     totals.Delayed,
		AvgDelay:    avgDelay,
		LateUpTo10:  totals.LateUpTo10,
		Late10To30:  totals.Late10To30,
		LateOver30:  totals.LateOver30,
		MaxDelay:    totals.MaxDelay,
	}
}

type periodTotals struct {
	byServiceType map[string]*DailyStats
	total         *DailyStats
}

// buildTimeseries sums the daily rows into periods; each period has one entry per service type plus TOTAL last.
// Periods without rows are left out.
func buildTimeseries(rows []DailyStats, granularity string) ([]TimeseriesPeriod, error) {
	totalsByPeriod := map[string]*periodTotals{}
	for _, row := range rows {
		day, err := time.Parse(dayLayout, row.Day)
		if err != nil {
			return nil, fmt.Errorf("parsing daily_stats day %q: %w", row.Day, err)
		}
		label := periodLabel(day, granularity)
		period := totalsByPeriod[label]
		if period == nil {
			period = &periodTotals{
				byServiceType: map[string]*DailyStats{},
				total:         &DailyStats{ServiceType: totalServiceType},
			}
			totalsByPeriod[label] = period
		}

		serviceTotals := period.byServiceType[row.ServiceType]
		if serviceTotals == nil {
			serviceTotals = &DailyStats{ServiceType: row.ServiceType}
			period.byServiceType[row.ServiceType] = serviceTotals
		}
		addDailyStats(serviceTotals, row)
		addDailyStats(period.total, row)
	}

	periods := []TimeseriesPeriod{}
	for _, label := range slices.Sorted(maps.Keys(totalsByPeriod)) {
		totals := totalsByPeriod[label]
		period := TimeseriesPeriod{Period: label, Stats: []TimeseriesStats{}}
		for _, serviceType := range slices.Sorted(maps.Keys(totals.byServiceType)) {
			period.Stats = append(period.Stats, toTimeseriesStats(*totals.byServiceType[serviceType]))
		}
		period.Stats = append(period.Stats, toTimeseriesStats(*totals.total))
		periods = append(periods, period)
	}
	return periods, nil
}

func getTimeseries(ctx context.Context, db *sql.DB, cache *dayCache, now time.Time, from, to time.Time, granularity string) (*TimeseriesResponse, error) {
	readStats := func(ctx context.Context, fromDay, toDay string) ([]DailyStats, error) {
		return selectDailyStats(ctx, db, fromDay, toDay)
	}
	rows, err := loadStats(ctx, cache, now, from, to, readStats)
	if err != nil {
		return nil, err
	}
	periods, err := buildTimeseries(rows, granularity)
	if err != nil {
		return nil, err
	}
	return &TimeseriesResponse{Granularity: granularity, Periods: periods}, nil
}

// topWorstTrips keeps the limit worst trips: delay DESC, then train number, then trip id so the order is stable.
func topWorstTrips(trips []DailyWorstTrip, limit int) []DailyWorstTrip {
	sorted := slices.Clone(trips)
	slices.SortFunc(sorted, func(a, b DailyWorstTrip) int {
		return cmp.Or(
			cmp.Compare(b.DelayMinutes, a.DelayMinutes),
			cmp.Compare(a.TrainNumber, b.TrainNumber),
			cmp.Compare(a.TripID, b.TripID),
		)
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

// serviceType "" means all types.
func getWorstTrips(ctx context.Context, db *sql.DB, cache *dayCache, now time.Time, from, to time.Time, serviceType string) ([]DailyWorstTrip, error) {
	readWorstTrips := func(ctx context.Context, fromDay, toDay string) ([]DailyWorstTrip, error) {
		return selectDailyWorstTrips(ctx, db, fromDay, toDay)
	}
	trips, err := loadWorstTrips(ctx, cache, now, from, to, readWorstTrips)
	if err != nil {
		return nil, err
	}

	candidates := []DailyWorstTrip{}
	for _, trip := range trips {
		if serviceType == "" || trip.ServiceType == serviceType {
			candidates = append(candidates, trip)
		}
	}

	worstTrips := topWorstTrips(candidates, worstTripsLimit)
	for i := range worstTrips {
		origin, found := shared.Stations[worstTrips[i].OriginStation]
		if found {
			worstTrips[i].OriginStationName = origin.Designation
		}
		destination, found := shared.Stations[worstTrips[i].DestinationStation]
		if found {
			worstTrips[i].DestinationStationName = destination.Designation
		}
	}
	return worstTrips, nil
}
