package main

import (
	"testing"
	"train-delays/shared"
)

func TestBuildTrip(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	completed := "COMPLETED"
	inTransit := "IN_TRANSIT"

	tests := []struct {
		name          string
		status        *string
		stops         []TrainStop
		wantReady     bool
		wantCancelled bool
	}{
		{
			name:   "completed trip is ready",
			status: &completed,
			stops: []TrainStop{
				{Station: stationStub("A"), Departure: strPtr("10:00"), ETD: strPtr("10:02")},
				{Station: stationStub("B"), Arrival: strPtr("11:00"), ETA: strPtr("11:05")},
			},
			wantReady: true,
		},
		{
			name:   "in-progress trip is not ready",
			status: &inTransit,
			stops: []TrainStop{
				{Station: stationStub("A"), Departure: strPtr("10:00")},
				{Station: stationStub("B"), Arrival: strPtr("11:00")},
			},
			wantReady: false,
		},
		{
			name:   "nil status, not cancelled, is not ready",
			status: nil,
			stops: []TrainStop{
				{Station: stationStub("A")},
				{Station: stationStub("B")},
			},
			wantReady: false,
		},
		{
			name:   "cancelled origin is ready even though not completed",
			status: &inTransit,
			stops: []TrainStop{
				{Station: stationStub("A"), Supression: &Supression{Code: "CANC"}},
				{Station: stationStub("B")},
			},
			wantReady:     true,
			wantCancelled: true,
		},
		{
			name:   "cancelled destination is ready even with nil status",
			status: nil,
			stops: []TrainStop{
				{Station: stationStub("A")},
				{Station: stationStub("B"), Supression: &Supression{Code: "CANC"}},
			},
			wantReady:     true,
			wantCancelled: true,
		},
		{
			name:      "fewer than two stops is never ready",
			status:    &completed,
			stops:     []TrainStop{{Station: stationStub("A")}},
			wantReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timetable := &TrainTimetable{Status: tt.status, TrainStops: tt.stops}

			trip, ready := buildTrip(123, timetable)
			if ready != tt.wantReady {
				t.Fatalf("buildTrip() ready = %v, want %v", ready, tt.wantReady)
			}
			if !ready {
				return
			}
			if trip.TrainNumber != 123 {
				t.Errorf("buildTrip() TrainNumber = %v, want 123", trip.TrainNumber)
			}
			if (trip.Supression != nil) != tt.wantCancelled {
				t.Errorf("buildTrip() cancelled = %v, want %v", trip.Supression != nil, tt.wantCancelled)
			}
		})
	}
}

func stationStub(code string) shared.StationInfo {
	return shared.StationInfo{Code: code}
}

func TestTripDay(t *testing.T) {
	const today, yesterday = "2026-04-05", "2026-04-04"
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name          string
		departureTime *string
		currentHour   int
		want          string
	}{
		{
			name:          "midnight-crossing: departed 23:50, polled at 01:00 the next day belongs to yesterday",
			departureTime: strPtr("23:50"),
			currentHour:   1,
			want:          yesterday,
		},
		{
			name:          "same-day: departure hour at or before current hour belongs to today",
			departureTime: strPtr("10:00"),
			currentHour:   11,
			want:          today,
		},
		{
			name:          "departure hour equal to current hour belongs to today",
			departureTime: strPtr("10:30"),
			currentHour:   10,
			want:          today,
		},
		{
			name:          "no departure time falls back to today",
			departureTime: nil,
			currentHour:   10,
			want:          today,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tripDay(tt.departureTime, today, yesterday, tt.currentHour)
			if got != tt.want {
				t.Errorf("tripDay() = %v, want %v", got, tt.want)
			}
		})
	}
}
