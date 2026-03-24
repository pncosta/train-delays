package main

import (
	"context"
	"fmt"
	"strings"
)

// 1 - Get all arrivals from all stations
// 2 - Store all those arrivals (and metadata) in DB
func foobarGetName(ctx context.Context, cpClient CPClient) error {

	stations, _ := cpClient.FetchStations(ctx)

	date := "2026-03-23" // TODO

	for _, station := range stations {

		trains, err := cpClient.FetchTimetable(ctx, station.Code, date)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}

		filterDestinationOnly := func(s Trip) bool { return strings.HasPrefix(s.TrainDestination.Code, station.Code) }
		filteredTrains := filter(trains.Trips, filterDestinationOnly)

		fmt.Printf("Found %d (from %d) trains that finish in station %s)\n", len(filteredTrains), len(trains.Trips), station)

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
