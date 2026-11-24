package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseDayRange(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"single day", "from=2026-01-01&to=2026-01-01", false},
		{"exactly the max span", "from=2023-01-01&to=2026-01-04", false},
		{"one day over the max span", "from=2023-01-01&to=2026-01-05", true},
	}
	for _, test := range tests {
		query, err := url.ParseQuery(test.query)
		if err != nil {
			t.Fatal(err)
		}
		from, to, err := parseDayRange(query)
		if (err != nil) != test.wantErr {
			t.Errorf("%s: err = %v, want error: %v", test.name, err, test.wantErr)
		}
		if err == nil && (from.Format(dayLayout) != query.Get("from") || to.Format(dayLayout) != query.Get("to")) {
			t.Errorf("%s: parsed %v..%v", test.name, from, to)
		}
	}
}

// Invalid requests are rejected before any DB access, so the handlers get a nil DB.
func TestHandlersRejectInvalidParams(t *testing.T) {
	handler := &Handler{}
	daily := handler.handleDailyStats(nil, newDayCache())
	worstTrips := handler.handleWorstTrips(nil, newDayCache())

	tests := []struct {
		name    string
		handler http.HandlerFunc
		url     string
	}{
		{"daily missing from", daily, "/api/stats/daily?to=2026-02-01"},
		{"daily malformed to", daily, "/api/stats/daily?from=2026-01-01&to=yesterday"},
		{"daily to before from", daily, "/api/stats/daily?from=2026-02-01&to=2026-01-01"},
		{"daily range too long", daily, "/api/stats/daily?from=2020-01-01&to=2026-12-31"},
		{"daily absurd range", daily, "/api/stats/daily?from=0001-01-01&to=9999-12-31"},
		{"worst trips missing to", worstTrips, "/api/stats/worst-trips?from=2026-01-01"},
		{"worst trips to before from", worstTrips, "/api/stats/worst-trips?from=2026-02-01&to=2026-01-01"},
		{"worst trips range too long", worstTrips, "/api/stats/worst-trips?from=2020-01-01&to=2026-12-31"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		test.handler(recorder, httptest.NewRequest(http.MethodGet, test.url, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", test.name, recorder.Code)
		}
	}
}
