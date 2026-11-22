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

func buildDailyStats(trips []trip) []dailyStats {
	sorted := slices.Clone(trips)
	slices.SortFunc(sorted, func(a, b trip) int {
		return cmp.Or(cmp.Compare(a.day(), b.day()), cmp.Compare(a.service(), b.service()))
	})
	var out []dailyStats
	for _, t := range sorted {
		if len(out) == 0 || out[len(out)-1].day != t.day() || out[len(out)-1].serviceType != t.service() {
			out = append(out, dailyStats{day: t.day(), serviceType: t.service()})
		}
		s := &out[len(out)-1]
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
		if d <= delayedThreshold(s.serviceType) {
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
	return out
}

// Worst trips are the delayed subset, so the list matches the delayed counter.
func buildWorstTrips(trips []trip) []worstTrip {
	var delayed []trip
	for _, t := range trips {
		if t.eligible() && t.delay() > delayedThreshold(t.service()) {
			delayed = append(delayed, t)
		}
	}
	slices.SortFunc(delayed, func(a, b trip) int {
		return cmp.Or(cmp.Compare(a.day(), b.day()), cmp.Compare(a.service(), b.service()), compareWorse(a, b))
	})
	var out []worstTrip
	rank := 0
	for i, t := range delayed {
		if i == 0 || t.day() != delayed[i-1].day() || t.service() != delayed[i-1].service() {
			rank = 0
		}
		rank++
		if rank <= worstTripsPerGroup {
			out = append(out, worstTrip{day: t.day(), serviceType: t.service(), rank: rank, trip: t, delay: t.delay()})
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

// rollupDay rolls up one day (YYYY-MM-DD); a day already rolled up is left untouched.
func rollupDay(ctx context.Context, db *sql.DB, day string) error {
	date, err := time.Parse(dayLayout, day)
	if err != nil {
		return fmt.Errorf("failed to parse day %q: %w", day, err)
	}
	nextDay := date.AddDate(0, 0, 1).Format(dayLayout)

	selected, err := selectTrips(ctx, db, day, nextDay)
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
	return insertRollups(ctx, db, buildDailyStats(trips), buildWorstTrips(trips))
}

// rollupYesterday rolls up yesterday once Lisbon time is past the overnight gap; reruns keep existing rows.
func rollupYesterday(ctx context.Context, db *sql.DB, nowLisbon time.Time) error {
	day, due := yesterdayToRollup(nowLisbon)
	if !due {
		return nil
	}
	return rollupDay(ctx, db, day)
}

// backfillRollups rolls up every complete day in trips, one day per transaction (yesterday only from rollupHour).
func backfillRollups(ctx context.Context, db *sql.DB, nowLisbon time.Time) error {
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

	// Before rollupHour yesterday isn't complete yet, so stop before it instead of before today.
	stopBeforeDate := nowLisbon
	if _, due := yesterdayToRollup(nowLisbon); !due {
		stopBeforeDate = nowLisbon.AddDate(0, 0, -1)
	}
	stopBeforeDay := stopBeforeDate.Format(dayLayout)

	day := firstTripDate
	for day.Format(dayLayout) < stopBeforeDay {
		if err := rollupDay(ctx, db, day.Format(dayLayout)); err != nil {
			return fmt.Errorf("backfill %s: %w", day.Format(dayLayout), err)
		}
		day = day.AddDate(0, 0, 1)
	}
	return nil
}
