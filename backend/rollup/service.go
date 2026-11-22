package main

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"log"
	"slices"
	"time"
)

const dayLayout = "2006-01-02"

// Late bands partition the delays above the delayed threshold.
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

// The id prefix is the day (ids are YYYY-MM-DD-<train>).
func (t trip) day() string { return t.id[:len(dayLayout)] }

func (t trip) service() string {
	if t.serviceType == nil {
		return ""
	}
	return *t.serviceType
}

func (t trip) group() dayService { return dayService{t.day(), t.service()} }

func (t trip) cancelled() bool { return t.isCancelled != nil && *t.isCancelled }

// Delay-eligible rows have an arrival side and aren't cancelled.
func (t trip) eligible() bool { return t.scheduledArrival != nil && !t.cancelled() }

// A NULL delay counts as 0.
func (t trip) delay() int {
	if t.delayMinutes == nil {
		return 0
	}
	return *t.delayMinutes
}

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

func buildDailyStats(trips []trip) []dailyStats {
	byGroup := map[dayService]*dailyStats{}
	for _, t := range trips {
		g := t.group()
		s := byGroup[g]
		if s == nil {
			s = &dailyStats{dayService: g}
			byGroup[g] = s
		}
		s.trips++
		if t.cancelled() {
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
func buildWorstTrips(trips []trip) []worstTrip {
	byGroup := map[dayService][]trip{}
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
			out = append(out, worstTrip{dayService: g, rank: i + 1, trip: t, delay: t.delay()})
		}
	}
	return out
}

// Delay DESC, train_number ASC (NULL first, as in SQL), id ASC.
func compareWorse(a, b trip) int {
	if c := cmp.Compare(b.delay(), a.delay()); c != 0 {
		return c
	}
	if (a.trainNumber == nil) != (b.trainNumber == nil) {
		if a.trainNumber == nil {
			return -1
		}
		return 1
	}
	if a.trainNumber != nil {
		if c := cmp.Compare(*a.trainNumber, *b.trainNumber); c != 0 {
			return c
		}
	}
	return cmp.Compare(a.id, b.id)
}

// yesterdayToRollup returns the day to roll up once Lisbon time is past the overnight gap.
func yesterdayToRollup(nowLisbon time.Time) (string, bool) {
	if nowLisbon.Hour() < rollupHour {
		return "", false
	}
	return nowLisbon.AddDate(0, 0, -1).Format(dayLayout), true
}

func lisbonNow(t time.Time) (time.Time, error) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load Europe/Lisbon: %w", err)
	}
	return t.In(lisbon), nil
}

// rollupRange replaces the rollup rows for days in [from, to) atomically; from and to are YYYY-MM-DD.
func rollupRange(ctx context.Context, db *sql.DB, from, to string) error {
	selected, err := selectTrips(ctx, db, from, to)
	if err != nil {
		return err
	}
	trips := make([]trip, 0, len(selected))
	for _, t := range selected {
		if len(t.id) < len(dayLayout) {
			log.Printf("skipping trip with malformed id %q", t.id)
			continue
		}
		trips = append(trips, t)
	}
	if len(trips) == 0 {
		return nil
	}
	return replaceRollups(ctx, db, from, to, buildDailyStats(trips), buildWorstTrips(trips))
}

func rollupDay(ctx context.Context, db *sql.DB, day string) error {
	d, err := time.Parse(dayLayout, day)
	if err != nil {
		return fmt.Errorf("failed to parse day %q: %w", day, err)
	}
	return rollupRange(ctx, db, day, d.AddDate(0, 0, 1).Format(dayLayout))
}

// rollupYesterdayIfDue costs one marker read when yesterday is already rolled up.
func rollupYesterdayIfDue(ctx context.Context, db *sql.DB, nowLisbon time.Time) error {
	day, due := yesterdayToRollup(nowLisbon)
	if !due {
		return nil
	}
	done, err := dailyStatsExist(ctx, db, day)
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	return rollupDay(ctx, db, day)
}

// backfillRollups rolls up every complete day in trips (yesterday only from rollupHour), one calendar
// month per transaction to keep each Turso transaction bounded.
func backfillRollups(ctx context.Context, db *sql.DB, nowLisbon time.Time) error {
	minID, found, err := selectMinTripID(ctx, db)
	if err != nil {
		return err
	}
	if !found || len(minID) < len(dayLayout) {
		return nil
	}
	firstDay := minID[:len(dayLayout)]
	first, err := time.Parse(dayLayout, firstDay)
	if err != nil {
		return fmt.Errorf("failed to parse first trip day %q: %w", minID, err)
	}

	end := nowLisbon
	if _, due := yesterdayToRollup(nowLisbon); !due {
		end = nowLisbon.AddDate(0, 0, -1)
	}
	until := end.Format(dayLayout)
	for start := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC); start.Format(dayLayout) < until; start = start.AddDate(0, 1, 0) {
		from := max(start.Format(dayLayout), firstDay)
		to := min(start.AddDate(0, 1, 0).Format(dayLayout), until)
		if err := rollupRange(ctx, db, from, to); err != nil {
			return fmt.Errorf("backfill %s..%s: %w", from, to, err)
		}
	}
	return nil
}
