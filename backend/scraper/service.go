package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// This function gets and stores the trips that finished in the last hour
//
// 1 - Get list of relevant train stations
// 2 - for each station, gets all the trips that finish in that station in the last hour
// (CP only keeps the delay info for a random(?) number of hours before it is removed, so we try to get the latest trains that just finished)
// 3 - Upsert all those trips in DB
func getAndStoreTrips(ctx context.Context, cpClient *CPClient, dbClient *DBClient) error {
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	stations, _ := cpClient.FetchStations(ctx)
	log.Printf("Getting trips since %v\n", oneHourAgo)

	for _, station := range stations {
		trips, err := cpClient.FetchTrips(ctx, station.Code, oneHourAgo)
		if err != nil {
			return err
		}

		filterTrips := func(t Trip) bool {
			// trip doesnt end in current station - filter out
			if !strings.HasPrefix(t.TrainDestination.Code, station.Code) {
				return false
			}

			// check trips that already finished or are finishing soon (in the next few minutes)
			// first use ETA, if ETA is nil use  ArrivalTime
			// TODO - check what happens at midnight - even CP API is weird with dates so gotta understand what happens end of the day
			soonish := time.Now().Add(10 * time.Minute)

			if t.ETA != nil {
				// Create a new time with ETA from trip and adding current day
				eta, _ := time.ParseInLocation("2006-01-02 15:04", time.Now().Format("2006-01-02")+" "+*t.ETA, time.Now().Location())
				if soonish.After(eta) {
					// trip is finished or about to finish - store already
					return true
				}
			}

			if t.ArrivalTime != nil {
				// Create a new time with ETA from trip and adding current day
				eta, _ := time.ParseInLocation("2006-01-02 15:04", time.Now().Format("2006-01-02")+" "+*t.ArrivalTime, time.Now().Location())
				if soonish.After(eta) {
					// trip is finished or about to finish - store already
					return true
				}
			}

			// either ETA and arrivaltime are nil OR both are too much in the future and trip can be igored
			return false

		}
		day := oneHourAgo.Format("2006-01-02")
		err = dbClient.InsertTrips(day, trips, filterTrips)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}

	}

	return nil
}
