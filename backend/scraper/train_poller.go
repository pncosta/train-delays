package main

import (
	"context"
	"fmt"
	"time"
)

var now = time.Now

// pollPaceDelay paces sequential per-train requests. CP rate-limits aggressively - live
// testing saw 429s even on unparallelized sequential calls - so this is deliberately
// conservative. At ~1950 trains this caps a full run at ~32 minutes, comfortably inside
// the conservatively-assumed 2-3h CP retention window regardless of cron frequency.
const pollPaceDelay = 1 * time.Second

// pollTrains polls CP's per-train timetable endpoint for every train CP runs, inserting
// whichever have reached a terminal state (completed or cancelled).
//
// There's deliberately no "already captured" precheck here: CP calls are free (just
// rate-limited, which pollPaceDelay already respects) while Turso DB reads are billed, so
// re-polling and redundantly re-upserting an already-captured train (a harmless no-op via
// ON CONFLICT DO UPDATE) is cheaper than spending a DB read to avoid it.
//
// 1 - Fetch the full list of train numbers CP runs
// 2 - Poll each one at a time, paced, collecting any that are now complete/cancelled
// 3 - Batch-write everything collected in one pass at the end
func pollTrains(ctx context.Context, cpClient *CPClient, dbClient *DBClient) error {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return fmt.Errorf("error loading time location: %w", err)
	}

	trains, err := cpClient.FetchTrains(ctx)
	if err != nil {
		return fmt.Errorf("error fetching train list: %w", err)
	}

	trips := map[string]Trip{}
	for i, t := range trains {
		trainNumber := t.TrainNumber

		if i > 0 {
			time.Sleep(pollPaceDelay)
		}

		// Re-derived from the current wall clock on every train, not a run-start
		// snapshot: a full run can take ~32 minutes, so by the time it reaches a given
		// train the calendar day may have rolled over since the run started.
		pollNow := now().In(lisbon)
		pollToday := pollNow.Format("2006-01-02")
		pollYesterday := pollNow.AddDate(0, 0, -1).Format("2006-01-02")

		timetable, err := cpClient.FetchTrainTimetable(ctx, trainNumber, pollToday)
		if err != nil && isCalendarInvalidDate(err) {
			// pollToday isn't a valid run date for this train (e.g. a weekday-only train
			// polled just after midnight, where today is now the day it doesn't run) -
			// retry once against yesterday before giving up on this train.
			time.Sleep(pollPaceDelay)
			timetable, err = cpClient.FetchTrainTimetable(ctx, trainNumber, pollYesterday)
		}
		if err != nil {
			fmt.Printf("error fetching timetable for train %d: %v\n", trainNumber, err)
			continue
		}

		trip, ready := buildTrip(trainNumber, timetable)
		if !ready {
			continue
		}

		day := tripDay(trip.DepartureTime, pollToday, pollYesterday, pollNow.Hour())
		trips[tripID(day, trainNumber)] = trip
	}

	if err := dbClient.InsertCompletedTrips(trips); err != nil {
		return fmt.Errorf("error saving trips: %w", err)
	}

	return nil
}

func tripID(day string, trainNumber int) string {
	return fmt.Sprintf("%s-%d", day, trainNumber)
}

// tripDay picks which calendar day a trip's row belongs to by comparing its departure
// hour to the hour it was polled at: a departure hour later than the current hour could
// only have happened yesterday, since it hasn't happened yet today. This is exact (no
// guessing which of several candidate days is "closest") because a trip is always polled
// shortly after CP marks it complete/cancelled, so its departure can only be today or
// yesterday relative to the poll - never further back.
func tripDay(departureTime *string, today, yesterday string, currentHour int) string {
	if departureTime == nil {
		return today
	}
	departure, err := time.Parse("15:04", *departureTime)
	if err != nil {
		return today
	}
	if departure.Hour() > currentHour {
		return yesterday
	}
	return today
}

// buildTrip extracts the origin/destination stops from a train's timetable and reports
// whether the trip has reached a terminal state worth writing: either CP marked it
// COMPLETED, or either bookend stop is cancelled (same supression signal the old
// per-station scraper used for is_cancelled) - a cancelled train may never reach
// COMPLETED, so cancellation is treated as terminal too.
func buildTrip(trainNumber int, timetable *TrainTimetable) (Trip, bool) {
	if len(timetable.TrainStops) < 2 {
		return Trip{}, false
	}

	origin := timetable.TrainStops[0]
	destination := timetable.TrainStops[len(timetable.TrainStops)-1]

	supression := origin.Supression
	if supression == nil {
		supression = destination.Supression
	}

	completed := timetable.Status != nil && *timetable.Status == "COMPLETED"
	if !completed && supression == nil {
		return Trip{}, false
	}

	return Trip{
		TrainNumber:      trainNumber,
		TrainService:     timetable.ServiceCode,
		TrainOrigin:      origin.Station,
		TrainDestination: destination.Station,
		DepartureTime:    origin.Departure,
		ETD:              origin.ETD,
		ArrivalTime:      destination.Arrival,
		ETA:              destination.ETA,
		Delay:            timetable.Delay,
		Supression:       supression,
	}, true
}
