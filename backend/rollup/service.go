package main

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"maps"
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

func (t trip) service() string {
	if t.serviceType == nil {
		return ""
	}
	return *t.serviceType
}

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

// buildDailyStats returns one row per service type, sorted by service type; all trips belong to day.
func buildDailyStats(day string, trips []trip) []dailyStats {
	statsByService := map[string]*dailyStats{}
	for _, t := range trips {
		service := t.service()
		stats := statsByService[service]
		if stats == nil {
			stats = &dailyStats{day: day, serviceType: service}
			statsByService[service] = stats
		}

		stats.trips++
		if t.cancelled() {
			stats.cancelled++
		}
		if !t.eligible() {
			continue
		}

		delay := t.delay()
		if stats.delayCount == 0 || delay > stats.maxDelay {
			stats.maxDelay = delay
		}
		stats.delayCount++
		stats.delaySum += delay

		if delay <= delayedThreshold(service) {
			stats.onTime++
			continue
		}
		stats.delayed++
		switch {
		case delay <= lateBandShortMax:
			stats.lateUpTo10++
		case delay <= lateBandMediumMax:
			stats.late10To30++
		default:
			stats.lateOver30++
		}
	}

	var out []dailyStats
	for _, service := range slices.Sorted(maps.Keys(statsByService)) {
		out = append(out, *statsByService[service])
	}
	return out
}

// buildWorstTrips returns the worst delayed trips per service type; all trips belong to day.
// Only delayed trips are ranked, so the list matches the delayed counter.
func buildWorstTrips(day string, trips []trip) []worstTrip {
	var delayed []trip
	for _, t := range trips {
		if t.eligible() && t.delay() > delayedThreshold(t.service()) {
			delayed = append(delayed, t)
		}
	}
	slices.SortFunc(delayed, func(a, b trip) int {
		return cmp.Or(cmp.Compare(a.service(), b.service()), compareWorse(a, b))
	})

	var out []worstTrip
	rank := 0
	for i, t := range delayed {
		if i == 0 || t.service() != delayed[i-1].service() {
			rank = 0
		}
		rank++
		if rank <= worstTripsPerGroup {
			out = append(out, worstTrip{day: day, serviceType: t.service(), rank: rank, trip: t, delay: t.delay()})
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

func lisbonNow(t time.Time) (time.Time, error) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load Europe/Lisbon: %w", err)
	}
	return t.In(lisbon), nil
}

// rollupDay rolls up one day (YYYY-MM-DD); a day already rolled up is left untouched.
func rollupDay(ctx context.Context, db *sql.DB, day string) error {
	date, err := time.Parse(dayLayout, day)
	if err != nil {
		return fmt.Errorf("failed to parse day %q: %w", day, err)
	}
	nextDay := date.AddDate(0, 0, 1).Format(dayLayout)

	trips, err := selectTrips(ctx, db, day, nextDay)
	if err != nil {
		return err
	}
	if len(trips) == 0 {
		return nil
	}
	return insertRollups(ctx, db, buildDailyStats(day, trips), buildWorstTrips(day, trips))
}

// backfillRollups rolls up every day from the first trip up to (not including) today, one day per transaction.
func backfillRollups(ctx context.Context, db *sql.DB, today string) error {
	minID, found, err := selectMinTripID(ctx, db)
	if err != nil {
		return err
	}
	if !found || len(minID) < len(dayLayout) {
		return nil
	}
	firstTripDay := minID[:len(dayLayout)]
	firstTripDate, err := time.Parse(dayLayout, firstTripDay)
	if err != nil {
		return fmt.Errorf("failed to parse first trip day %q: %w", minID, err)
	}

	day := firstTripDate
	for day.Format(dayLayout) < today {
		if err := rollupDay(ctx, db, day.Format(dayLayout)); err != nil {
			return fmt.Errorf("backfill %s: %w", day.Format(dayLayout), err)
		}
		day = day.AddDate(0, 0, 1)
	}
	return nil
}
