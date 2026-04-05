package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
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

	// Set location to lisbon - needed to have correct input for CP API
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		fmt.Printf("error loading time location: %v", err)
	}

	nowLisbon := now().In(lisbon)
	oneHourAgo := nowLisbon.Add(-1 * time.Hour)
	day := oneHourAgo.Format("2006-01-02")
	stations, _ := cpClient.FetchStations(ctx)
	log.Printf("Getting trips since %v\n", oneHourAgo)

	for _, station := range stations {
		trips, err := cpClient.FetchTrips(ctx, station.Code, oneHourAgo)
		if err != nil {
			return err
		}

		// Filter out trips that START in current station - from those we want to store the staring time
		startingTrips := filterStartingTrips(trips, nowLisbon, station.Code)
		err = dbClient.InsertStartingTrips(day, startingTrips)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}

		endingTrips := filterEndingTrips(trips, nowLisbon, station.Code)
		err = dbClient.InsertEndingTrips(day, endingTrips)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}
	}

	return nil
}

func filterStartingTrips(trips []Trip, now time.Time, originStation string) []Trip {
	startingTrips := Filter(trips, func(t Trip) bool {
		if !strings.HasPrefix(t.TrainOrigin.Code, originStation) {
			return false
		}
		if t.DepartureTime != nil {
			departure, err := time.ParseInLocation("2006-01-02 15:04", now.Format("2006-01-02")+" "+*t.DepartureTime, now.Location())
			if err != nil {
				return false
			}

			windowStart := now.Add(-1 * 30 * time.Minute)
			windowEnd := now.Add(15 * time.Minute)
			if departure.After(windowStart) && departure.Before(windowEnd) {
				return true
			}
		}
		return false
	})
	return startingTrips
}

func filterEndingTrips(trips []Trip, now time.Time, destinationStation string) []Trip {
	startingTrips := Filter(trips, func(t Trip) bool {
		// trip doesnt end in current station - filter out
		if !strings.HasPrefix(t.TrainDestination.Code, destinationStation) {
			return false
		}

		// check trips that already finished or are finishing soon (in the next few minutes)
		// first use ETA, if ETA is nil use  ArrivalTime
		// TODO - check what happens at midnight and if it is relevant or not
		windowStart := now.Add(-1 * 30 * time.Minute)
		windowEnd := now.Add(15 * time.Minute)

		if t.ETA != nil {
			// Create a new time with ETA from trip and adding current day
			eta, err := time.ParseInLocation("2006-01-02 15:04", now.Format("2006-01-02")+" "+*t.ETA, now.Location())
			if err != nil {
				return false
			}
			if eta.After(windowStart) && eta.Before(windowEnd) {
				return true
			}
		}

		if t.ArrivalTime != nil {
			// Create a new time with ETA from trip and adding current day
			eta, err := time.ParseInLocation("2006-01-02 15:04", now.Format("2006-01-02")+" "+*t.ArrivalTime, now.Location())
			if err != nil {
				return false
			}
			if eta.After(windowStart) && eta.Before(windowEnd) {
				return true
			}
		}

		// either ETA and arrivaltime are nil OR both are too much in the future and trip can be igored
		return false
	})
	return startingTrips
}
