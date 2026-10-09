package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"train-delays/shared"
)

var (
	now = time.Now
)

// This function gets and stores the trips that finished in the last hour
//
// 1 - Get list of relevant train stations
// 2 - for each station, gets all the trips that finish in that station in the last hour
// (CP only keeps the delay info for a random(?) number of hours before it is removed, so we try to get the latest trains that just finished)
// 3 - Upsert all those trips in DB
func getAndStoreTrips(ctx context.Context, cpClient *CPClient, dbClient *DBClient) error {
	// Set location to lisbon - needed to have correct hour input for CP API
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		fmt.Printf("error loading time location: %v", err)
	}

	nowLisbon := now().In(lisbon)
	oneHourAgo := nowLisbon.Add(-1 * time.Hour)
	for _, station := range shared.Stations {
		trips, err := cpClient.FetchTrips(ctx, station.Code, oneHourAgo)
		if err != nil {
			return err
		}
		// Filter out trips that START in current station - from those we want to store the staring time
		startingTrips := filterStartingTrips(trips, nowLisbon, station.Code)
		err = dbClient.InsertStartingTrips(nowLisbon, startingTrips)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}

		// Filter out trips that END in current station - from those we want to store all the other data
		endingTrips := filterEndingTrips(trips, nowLisbon, station.Code)
		err = dbClient.InsertEndingTrips(nowLisbon, endingTrips)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}
	}

	return nil
}

// filters out trip that start in the given originStation
// and whose departing hour was few minutes ago or in the next minutes
func filterStartingTrips(trips []Trip, now time.Time, originStation string) []Trip {
	windowStart := now.Add(-30 * time.Minute)
	windowEnd := now.Add(15 * time.Minute)

	return Filter(trips, func(t Trip) bool {
		if !strings.HasPrefix(t.TrainOrigin.Code, originStation) {
			return false
		}

		if t.DepartureTime == nil {
			return false
		}

		departure, err := resolveClockTime(*t.DepartureTime, now)
		if err != nil {
			return false
		}

		return departure.After(windowStart) && departure.Before(windowEnd)
	})
}

// filters out trip that end in the given destinationStation
// and whose arrival hour was few minutes ago or in the next minutes
func filterEndingTrips(trips []Trip, now time.Time, destinationStation string) []Trip {
	windowStart := now.Add(-30 * time.Minute)
	windowEnd := now.Add(15 * time.Minute)

	return Filter(trips, func(t Trip) bool {
		if !strings.HasPrefix(t.TrainDestination.Code, destinationStation) {
			return false
		}

		if t.ETA != nil {
			if eta, err := resolveClockTime(*t.ETA, now); err == nil {
				if eta.After(windowStart) && eta.Before(windowEnd) {
					return true
				}
			}
		}

		if t.ArrivalTime != nil {
			if arrival, err := resolveClockTime(*t.ArrivalTime, now); err == nil {
				if arrival.After(windowStart) && arrival.Before(windowEnd) {
					return true
				}
			}
		}

		// either ETA and arrivaltime are nil OR both are too much in the future and trip can be igored
		return false
	})
}
