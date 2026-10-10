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

func TestIsCaptured(t *testing.T) {
	const today, yesterday = "2026-10-10", "2026-10-09"

	tests := []struct {
		name     string
		captured map[string]bool
		want     bool
	}{
		{
			name:     "today's id captured",
			captured: map[string]bool{tripID(today, 850): true},
			want:     true,
		},
		{
			name:     "yesterday's id captured (FindCapturedIDs already applied the recency filter)",
			captured: map[string]bool{tripID(yesterday, 850): true},
			want:     true,
		},
		{
			name:     "neither id captured",
			captured: map[string]bool{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCaptured(850, today, yesterday, tt.captured)
			if got != tt.want {
				t.Errorf("isCaptured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCandidateIDs(t *testing.T) {
	const today, yesterday = "2026-10-10", "2026-10-09"
	trains := []TrainListEntry{{TrainNumber: 850}, {TrainNumber: 851}}

	todayIDs, yesterdayIDs := candidateIDs(trains, today, yesterday)

	wantToday := []string{tripID(today, 850), tripID(today, 851)}
	wantYesterday := []string{tripID(yesterday, 850), tripID(yesterday, 851)}

	for i, id := range wantToday {
		if todayIDs[i] != id {
			t.Errorf("candidateIDs() todayIDs[%d] = %v, want %v", i, todayIDs[i], id)
		}
	}
	for i, id := range wantYesterday {
		if yesterdayIDs[i] != id {
			t.Errorf("candidateIDs() yesterdayIDs[%d] = %v, want %v", i, yesterdayIDs[i], id)
		}
	}
}
