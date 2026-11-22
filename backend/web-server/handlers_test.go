package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Invalid requests are rejected before any DB access, so the handlers get a nil DB.
func TestHandlersRejectInvalidParams(t *testing.T) {
	handler := &Handler{}
	timeseries := handler.handleTimeseries(nil, newDayCache())
	worstTrips := handler.handleWorstTrips(nil, newDayCache())

	tests := []struct {
		name    string
		handler http.HandlerFunc
		url     string
	}{
		{"unknown granularity", timeseries, "/api/stats/timeseries?granularity=year&from=2026-01-01&to=2026-02-01"},
		{"missing granularity", timeseries, "/api/stats/timeseries?from=2026-01-01&to=2026-02-01"},
		{"missing from", timeseries, "/api/stats/timeseries?granularity=day&to=2026-02-01"},
		{"malformed to", timeseries, "/api/stats/timeseries?granularity=day&from=2026-01-01&to=yesterday"},
		{"to before from", timeseries, "/api/stats/timeseries?granularity=day&from=2026-02-01&to=2026-01-01"},
		{"too many day points", timeseries, "/api/stats/timeseries?granularity=day&from=2026-01-01&to=2026-12-31"},
		{"absurd range", timeseries, "/api/stats/timeseries?granularity=month&from=0001-01-01&to=9999-12-31"},
		{"too many month points", timeseries, "/api/stats/timeseries?granularity=month&from=2010-01-01&to=2026-12-31"},
		{"worst trips missing to", worstTrips, "/api/stats/worst-trips?from=2026-01-01"},
		{"worst trips to before from", worstTrips, "/api/stats/worst-trips?from=2026-02-01&to=2026-01-01"},
		{"worst trips range too long", worstTrips, "/api/stats/worst-trips?from=2000-01-01&to=2026-12-31"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		test.handler(recorder, httptest.NewRequest(http.MethodGet, test.url, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", test.name, recorder.Code)
		}
	}
}
