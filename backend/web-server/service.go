package main

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"sync"
	"time"
	"train-delays/shared"
)

const (
	dayLayout = "2006-01-02"

	totalSystemServiceType = "TOTAL_SYSTEM"

	urbanServiceType      = "U"
	urbanDelayedMinutes   = 3
	defaultDelayedMinutes = 5

	// about 3 years: bounds a single read, and the cache keeps repeat reads cheap
	maxRangeDays = 1100

	worstTripsLimit = 10
)

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

// Like the rollup: every trip counts in total, cancelled ones in cancelled, and only
// non-cancelled trips with a scheduled arrival take part in the delay stats (NULL delay counts as 0).
func (totals *summaryTotals) add(group summaryGroup) {
	totals.stats.TotalTrips += group.Count

	switch {
	case group.IsCancelled:
		totals.stats.CancelledCount += group.Count
	case !group.HasScheduledArrival:
		// not delay-eligible
	default:
		delay := 0
		if group.DelayMinutes != nil {
			delay = *group.DelayMinutes
		}
		totals.delaySum += delay * group.Count
		totals.delayCount += group.Count

		// delayed means strictly above the threshold
		if delay > delayedThreshold(group.ServiceType) {
			totals.stats.DelayedCount += group.Count
		} else {
			totals.stats.OnTimeCount += group.Count
		}
	}
}

func (totals *summaryTotals) toServiceStats() ServiceStats {
	stats := totals.stats
	if totals.delayCount > 0 {
		stats.AvgDelay = float64(totals.delaySum) / float64(totals.delayCount)
	}
	return stats
}

func summarizeGroups(groups []summaryGroup) *DashboardResponse {
	totalsByType := map[string]*summaryTotals{}
	systemTotals := &summaryTotals{stats: ServiceStats{ServiceType: totalSystemServiceType}}

	for _, group := range groups {
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

func daysBetween(from, to time.Time) []time.Time {
	var days []time.Time
	day := from
	for !day.After(to) {
		days = append(days, day)
		day = day.AddDate(0, 0, 1)
	}
	return days
}

// dayCache holds rows per day, including days with no rows; a cached day is never re-read from the DB.
// Each table has its own mutex, held during that table's read so concurrent requests for the same days read once;
// cached hits also wait while that read is in flight.
type dayCache struct {
	statsMutex      sync.Mutex
	stats           map[string][]DailyStats
	worstTripsMutex sync.Mutex
	worstTrips      map[string][]DailyWorstTrip
}

func newDayCache() *dayCache {
	return &dayCache{
		stats:      map[string][]DailyStats{},
		worstTrips: map[string][]DailyWorstTrip{},
	}
}

// Yesterday and today aren't cacheable: the rollup writes yesterday some time after 04:00 Lisbon.
// UTC is safe here because every day before UTC-yesterday was rolled up long ago, wherever "today" starts.
func isCacheableDay(day time.Time, now time.Time) bool {
	utcNow := now.UTC()
	yesterday := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day()-1, 0, 0, 0, 0, time.UTC)
	return day.Before(yesterday)
}

type statsReader func(ctx context.Context, fromDay, toDay string) ([]DailyStats, error)

type worstTripsReader func(ctx context.Context, fromDay, toDay string) ([]DailyWorstTrip, error)

// loadStats returns the daily_stats rows in [from, to]: cached days are served from the cache and
// everything from the first uncached day on is read in one query.
func loadStats(ctx context.Context, cache *dayCache, now time.Time, from, to time.Time, readStats statsReader) ([]DailyStats, error) {
	days := daysBetween(from, to)

	cache.statsMutex.Lock()
	defer cache.statsMutex.Unlock()

	firstUncached := len(days)
	for i, day := range days {
		_, cached := cache.stats[day.Format(dayLayout)]
		if !cached {
			firstUncached = i
			break
		}
	}

	rows := []DailyStats{}
	for _, day := range days[:firstUncached] {
		rows = append(rows, cache.stats[day.Format(dayLayout)]...)
	}
	if firstUncached == len(days) {
		return rows, nil
	}

	fresh, err := readStats(ctx, days[firstUncached].Format(dayLayout), to.Format(dayLayout))
	if err != nil {
		return nil, err
	}
	freshByDay := map[string][]DailyStats{}
	for _, row := range fresh {
		freshByDay[row.Day] = append(freshByDay[row.Day], row)
	}
	for _, day := range days[firstUncached:] {
		if isCacheableDay(day, now) {
			cache.stats[day.Format(dayLayout)] = freshByDay[day.Format(dayLayout)]
		}
	}
	return append(rows, fresh...), nil
}

// loadWorstTrips is loadStats for daily_worst_trips.
func loadWorstTrips(ctx context.Context, cache *dayCache, now time.Time, from, to time.Time, readWorstTrips worstTripsReader) ([]DailyWorstTrip, error) {
	days := daysBetween(from, to)

	cache.worstTripsMutex.Lock()
	defer cache.worstTripsMutex.Unlock()

	firstUncached := len(days)
	for i, day := range days {
		_, cached := cache.worstTrips[day.Format(dayLayout)]
		if !cached {
			firstUncached = i
			break
		}
	}

	trips := []DailyWorstTrip{}
	for _, day := range days[:firstUncached] {
		trips = append(trips, cache.worstTrips[day.Format(dayLayout)]...)
	}
	if firstUncached == len(days) {
		return trips, nil
	}

	fresh, err := readWorstTrips(ctx, days[firstUncached].Format(dayLayout), to.Format(dayLayout))
	if err != nil {
		return nil, err
	}
	freshByDay := map[string][]DailyWorstTrip{}
	for _, trip := range fresh {
		freshByDay[trip.Day] = append(freshByDay[trip.Day], trip)
	}
	for _, day := range days[firstUncached:] {
		if isCacheableDay(day, now) {
			cache.worstTrips[day.Format(dayLayout)] = freshByDay[day.Format(dayLayout)]
		}
	}
	return append(trips, fresh...), nil
}

func getDailyStats(ctx context.Context, dbClient *DBClient, cache *dayCache, now time.Time, from, to time.Time) ([]DailyStats, error) {
	return loadStats(ctx, cache, now, from, to, dbClient.SelectDailyStats)
}

// topWorstTrips keeps the limit worst trips: delay DESC, then train number, then trip id so the order is stable.
func topWorstTrips(trips []DailyWorstTrip, serviceType string, limit int) []DailyWorstTrip {
	candidates := []DailyWorstTrip{}
	for _, trip := range trips {
		if serviceType == "" || trip.ServiceType == serviceType {
			candidates = append(candidates, trip)
		}
	}
	slices.SortFunc(candidates, func(a, b DailyWorstTrip) int {
		return cmp.Or(
			cmp.Compare(b.DelayMinutes, a.DelayMinutes),
			cmp.Compare(a.TrainNumber, b.TrainNumber),
			cmp.Compare(a.TripID, b.TripID),
		)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// serviceType "" means all types.
func getWorstTrips(ctx context.Context, dbClient *DBClient, cache *dayCache, now time.Time, from, to time.Time, serviceType string) ([]DailyWorstTrip, error) {
	trips, err := loadWorstTrips(ctx, cache, now, from, to, dbClient.SelectDailyWorstTrips)
	if err != nil {
		return nil, err
	}

	worstTrips := topWorstTrips(trips, serviceType, worstTripsLimit)
	for i := range worstTrips {
		trip := &worstTrips[i]
		origin, found := shared.Stations[trip.OriginStation]
		if found {
			trip.OriginStationName = origin.Designation
		}
		destination, found := shared.Stations[trip.DestinationStation]
		if found {
			trip.DestinationStationName = destination.Designation
		}
	}
	return worstTrips, nil
}
