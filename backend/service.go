package main

import (
	"context"
	"fmt"
	"strings"
)

// 1 - Get list of relevant train stations
// 2 - for each one, gets all the trips that finish in that station
// 3 - Store all those trips
func getAndStoreTrips(ctx context.Context, cpClient CPClient, date string) error {
	stations, _ := cpClient.FetchStations(ctx)

	for _, station := range stations {
		trains, err := cpClient.FetchTimetable(ctx, station.Code, date)
		if err != nil {
			return err
		}

		filterDestinationOnly := func(s Trip) bool { return strings.HasPrefix(s.TrainDestination.Code, station.Code) }
		filteredTrains := filter(trains.Trips, filterDestinationOnly)

		fmt.Printf("%s - Found %d (from %d) trains that finish in station %s)\n", date, len(filteredTrains), len(trains.Trips), station)

		err = storeTrips(date, filteredTrains)
		if err != nil {
			fmt.Printf("error saving trips: %v", err)
		}

	}

	return nil
}

func storeTrips(date string, trips []Trip) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, trip := range trips {
		err = SaveTrip(tx, date, trip)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
