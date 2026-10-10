package main

import (
	"testing"
	"time"
)

func TestResolveArrivalDay(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Lisbon")
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name string
		trip Trip
		now  time.Time
		want string
	}{
		{
			name: "prefers ETA over ArrivalTime",
			trip: Trip{
				ArrivalTime: strPtr("23:50"),
				ETA:         strPtr("00:05"),
			},
			now:  time.Date(2026, 4, 5, 0, 10, 0, 0, loc),
			want: "2026-04-05",
		},
		{
			name: "falls back to now's day when neither clock is present",
			trip: Trip{},
			now:  time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			want: "2026-04-05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveArrivalDay(tt.trip, tt.now)
			if got != tt.want {
				t.Errorf("resolveArrivalDay() = %v, want %v", got, tt.want)
			}
		})
	}
}
