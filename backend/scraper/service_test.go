package main

import (
	"testing"
	"time"
)

func TestFilterStartingTrips(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Lisbon")
	// 2026-04-05 10:00 AM
	now := time.Date(2026, 4, 5, 10, 0, 0, 0, loc)

	// Helper to create a string pointer
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
			originStation:          "LISBON",
			givenTripDepartureTime: "10:10",
			givenTripOrigin:        "LISBON",
			wantCount:              1,
		},
		{
			name:                   "Departure 10 min ago",
			originStation:          "LISBON",
			givenTripDepartureTime: "09:50",
			givenTripOrigin:        "LISBON",
			wantCount:              1,
		},
		{
			name:                   "Now is too early (2 hours before departure)",
			originStation:          "LISBON",
			givenTripDepartureTime: "12:05",
			givenTripOrigin:        "LISBON",
			wantCount:              0,
		},
		{
			name:                   "Now is too late (2h after departure)",
			originStation:          "LISBON",
			givenTripDepartureTime: "08:00",
			givenTripOrigin:        "LISBON",
			wantCount:              0,
		},
		{
			name:                   "Wrong Station: Correct time but wrong origin",
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

			got := filterStartingTrips(trips, now, tt.originStation)

			if len(got) != tt.wantCount {
				t.Errorf("filterStartingTrips() got %v trips, want %v", len(got), tt.wantCount)
			}
		})
	}
}
