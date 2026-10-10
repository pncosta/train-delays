package main

import (
	"testing"
	"time"
)

func TestResolveClockTime(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Lisbon")

	tests := []struct {
		name    string
		now     time.Time
		clock   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "same day, no ambiguity",
			now:   time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			clock: "10:05",
			want:  time.Date(2026, 4, 5, 10, 5, 0, 0, loc),
		},
		{
			name:  "just after midnight, clock belongs to yesterday",
			now:   time.Date(2026, 4, 5, 0, 5, 0, 0, loc),
			clock: "23:50",
			want:  time.Date(2026, 4, 4, 23, 50, 0, 0, loc),
		},
		{
			name:  "just before midnight, clock belongs to tomorrow",
			now:   time.Date(2026, 4, 5, 23, 55, 0, 0, loc),
			clock: "00:05",
			want:  time.Date(2026, 4, 6, 0, 5, 0, 0, loc),
		},
		{
			name:    "invalid clock string",
			now:     time.Date(2026, 4, 5, 10, 0, 0, 0, loc),
			clock:   "not-a-time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveClockTime(tt.clock, tt.now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveClockTime() expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveClockTime() unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("resolveClockTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
