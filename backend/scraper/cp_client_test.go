package main

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIsCalendarInvalidDate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "500 status error is calendar-invalid",
			err:  &apiStatusError{StatusCode: http.StatusInternalServerError, Body: "Train 123 not valid for date 2026-04-05"},
			want: true,
		},
		{
			name: "500 status error wrapped by fmt.Errorf is still detected",
			err:  fmt.Errorf("fetching timetable for train %d: %w", 123, &apiStatusError{StatusCode: http.StatusInternalServerError}),
			want: true,
		},
		{
			name: "other status codes are not calendar-invalid",
			err:  &apiStatusError{StatusCode: http.StatusTooManyRequests},
			want: false,
		},
		{
			name: "non-apiStatusError is not calendar-invalid",
			err:  fmt.Errorf("http error: timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCalendarInvalidDate(tt.err); got != tt.want {
				t.Errorf("isCalendarInvalidDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
