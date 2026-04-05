package main

import (
	"testing"
	"time"
)

func TestFilterEndingTrips(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Lisbon")
	strPtr := func(s string) *string { return &s }
	tests := []struct {
		name                 string
		now                  time.Time
		destinationStation   string
		givenTripArrivalTime string
		givenTripDestination string
		wantCount            int
	}{
		{
			name:                 "Departure in 10 min",
			now:                  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			destinationStation:   "LISBON",
			givenTripArrivalTime: "10:10",
			givenTripDestination: "LISBON",
			wantCount:            1,
		},
		{
			name:                 "Departure 10 min ago",
			now:                  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			destinationStation:   "LISBON",
			givenTripArrivalTime: "09:50",
			givenTripDestination: "LISBON",
			wantCount:            1,
		},
		{
			name:                 "Now is too early (2 hours before departure)",
			now:                  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			destinationStation:   "LISBON",
			givenTripArrivalTime: "12:05",
			givenTripDestination: "LISBON",
			wantCount:            0,
		},
		{
			name:                 "Now is too late (2h after departure)",
			now:                  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			destinationStation:   "LISBON",
			givenTripArrivalTime: "08:00",
			givenTripDestination: "LISBON",
			wantCount:            0,
		},
		{
			name:                 "Wrong Station: Correct time but wrong origin",
			now:                  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			destinationStation:   "PORTO",
			givenTripArrivalTime: "10:05",
			givenTripDestination: "LISBON-A",
			wantCount:            0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock data
			trips := []Trip{
				{
					TrainDestination: StationInfo{Code: tt.givenTripDestination},
					ArrivalTime:      strPtr(tt.givenTripArrivalTime),
				},
			}

			got := filterEndingTrips(trips, tt.now, tt.destinationStation)

			if len(got) != tt.wantCount {
				t.Errorf("filterEndingTrips() got %v trips, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestFilterStartingTrips(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Lisbon")
	strPtr := func(s string) *string { return &s }
	tests := []struct {
		name                   string
		now                    time.Time
		originStation          string
		givenTripDepartureTime string
		givenTripOrigin        string
		wantCount              int
	}{
		{
			name:                   "Departure in 10 min",
			now:                    time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			originStation:          "LISBON",
			givenTripDepartureTime: "10:10",
			givenTripOrigin:        "LISBON",
			wantCount:              1,
		},
		{
			name:                   "Departure 10 min ago",
			now:                    time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			originStation:          "LISBON",
			givenTripDepartureTime: "09:50",
			givenTripOrigin:        "LISBON",
			wantCount:              1,
		},
		{
			name:                   "Now is too early (2 hours before departure)",
			now:                    time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			originStation:          "LISBON",
			givenTripDepartureTime: "12:05",
			givenTripOrigin:        "LISBON",
			wantCount:              0,
		},
		{
			name:                   "Now is too late (2h after departure)",
			now:                    time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			originStation:          "LISBON",
			givenTripDepartureTime: "08:00",
			givenTripOrigin:        "LISBON",
			wantCount:              0,
		},
		{
			name:                   "Wrong Station: Correct time but wrong origin",
			now:                    time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			originStation:          "PORTO",
			givenTripDepartureTime: "10:05",
			givenTripOrigin:        "LISBON-A",
			wantCount:              0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock data
			trips := []Trip{
				{
					TrainOrigin:   StationInfo{Code: tt.givenTripOrigin},
					DepartureTime: strPtr(tt.givenTripDepartureTime),
				},
			}

			got := filterStartingTrips(trips, tt.now, tt.originStation)

			if len(got) != tt.wantCount {
				t.Errorf("filterStartingTrips() got %v trips, want %v", len(got), tt.wantCount)
			}
		})
	}
}
