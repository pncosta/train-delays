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

// yesterdayRecencyWindow bounds how old a yesterday-id capture can be and still count as
// a genuine midnight-spanning capture rather than an unrelated ~24h-old run of the same
// recurring train number. A real midnight-spanning capture is always recent (written by
// the run that just observed it), while a same-recurring-train match from the day before
// is always close to 24h old - 4h is comfortably wide enough to cover any gap between
// cron runs plus the retention-window overlap, while nowhere near 24h.
const yesterdayRecencyWindow = 4 * time.Hour

// pollTrains polls CP's per-train timetable endpoint for every train not yet captured,
// inserting whichever have reached a terminal state (completed or cancelled).
//
// 1 - Fetch the full list of train numbers CP runs
// 2 - Drop whichever are already fully captured (today, or recently under yesterday's id for a late-night arrival captured just after midnight)
// 3 - Poll the rest one at a time, paced, collecting any that are now complete/cancelled, and batch-write them all at the end
func pollTrains(ctx context.Context, cpClient *CPClient, dbClient *DBClient) error {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return fmt.Errorf("error loading time location: %w", err)
	}
	nowLisbon := now().In(lisbon)
	today := nowLisbon.Format("2006-01-02")
	yesterday := nowLisbon.AddDate(0, 0, -1).Format("2006-01-02")
	recentSince := nowLisbon.Add(-yesterdayRecencyWindow).UTC().Format("2006-01-02 15:04:05")

	trains, err := cpClient.FetchTrains(ctx)
	if err != nil {
		return fmt.Errorf("error fetching train list: %w", err)
	}

	todayIDs, yesterdayIDs := candidateIDs(trains, today, yesterday)
	captured, err := dbClient.FindCapturedIDs(todayIDs, yesterdayIDs, recentSince)
	if err != nil {
		return fmt.Errorf("error checking already-captured trains: %w", err)
	}

	var trips []Trip
	polled := 0
	for _, t := range trains {
		trainNumber := t.TrainNumber
		if isCaptured(trainNumber, today, yesterday, captured) {
			continue
		}

		if polled > 0 {
			time.Sleep(pollPaceDelay)
		}
		polled++

		// Re-derived from the current wall clock on every train, not the run-start
		// snapshot above: a full run can take ~32 minutes, so by the time it reaches a
		// given train the calendar day may have rolled over since today/yesterday were
		// first computed.
		pollToday := now().In(lisbon).Format("2006-01-02")
		timetable, err := cpClient.FetchTrainTimetable(ctx, trainNumber, pollToday)
		if err != nil && isCalendarInvalidDate(err) {
			// pollToday isn't a valid run date for this train (e.g. a weekday-only train
			// polled just after midnight, where today is now the day it doesn't run) -
			// retry once against yesterday before giving up on this train.
			time.Sleep(pollPaceDelay)
			pollYesterday := now().In(lisbon).AddDate(0, 0, -1).Format("2006-01-02")
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
		trips = append(trips, trip)
	}

	if err := dbClient.InsertCompletedTrips(nowLisbon, trips); err != nil {
		return fmt.Errorf("error saving trips: %w", err)
	}

	return nil
}

func tripID(day string, trainNumber int) string {
	return fmt.Sprintf("%s-%d", day, trainNumber)
}

// candidateIDs builds the today/yesterday precheck ids for each train - kept as separate
// groups because FindCapturedIDs applies a recency filter to yesterdayIDs only.
func candidateIDs(trains []TrainListEntry, today, yesterday string) (todayIDs, yesterdayIDs []string) {
	todayIDs = make([]string, len(trains))
	yesterdayIDs = make([]string, len(trains))
	for i, t := range trains {
		todayIDs[i] = tripID(today, t.TrainNumber)
		yesterdayIDs[i] = tripID(yesterday, t.TrainNumber)
	}
	return todayIDs, yesterdayIDs
}

// isCaptured reports whether trainNumber already has a terminal row per the ids
// FindCapturedIDs returned as captured - the recency filter on the yesterday id is
// already baked into that result, so this just checks presence for either id.
func isCaptured(trainNumber int, today, yesterday string, captured map[string]bool) bool {
	return captured[tripID(today, trainNumber)] || captured[tripID(yesterday, trainNumber)]
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
