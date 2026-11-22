package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Handler struct {
}

func (h *Handler) handleSummary(_ context.Context, _ string, dbClient *DBClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysParam := r.URL.Query().Get("days")
		if daysParam == "" {
			daysParam = "7" // Default to last week
		}

		result, err := dbClient.GetSummary()
		if err != nil {
			log.Printf("error getting summary: %v", err)

		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func (h *Handler) handleWorstDelays(ctx context.Context, _ string, dbClient *DBClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysParam := r.URL.Query().Get("days")
		days, err := strconv.Atoi(daysParam)
		if err != nil || daysParam == "" {
			days = 7 // Default to last week
		}

		limit := 20 // Default limit
		result, err := dbClient.GetWorstlDelays(ctx, days, limit)
		if err != nil {
			log.Printf("error getting worst delays: %v", err)

		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func (h *Handler) handleCancellations(ctx context.Context, _ string, dbClient *DBClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysParam := r.URL.Query().Get("days")
		days, err := strconv.Atoi(daysParam)
		if err != nil || daysParam == "" {
			days = 7 // Default to last week
		}

		limit := 20 // Default limit
		summary, err := dbClient.GetCancellations(ctx, days, limit)
		if err != nil {
			log.Printf("error getting cancellations: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}
}

func (h *Handler) handleWorstAverageDelays(ctx context.Context, _ string, dbClient *DBClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysParam := r.URL.Query().Get("days")
		days, err := strconv.Atoi(daysParam)
		if err != nil || daysParam == "" {
			days = 7 // Default to last week
		}

		limit := 20 // Default limit
		summary, err := dbClient.GetWorstAverageDelays(ctx, days, limit)
		if err != nil {
			log.Printf("error getting worst average delays: %v", err)

		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}
}

// parseDayRange reads the required from and to query params; both are inclusive YYYY-MM-DD days.
func parseDayRange(query url.Values) (time.Time, time.Time, error) {
	from, err := time.Parse(dayLayout, query.Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be a YYYY-MM-DD day: %w", err)
	}
	to, err := time.Parse(dayLayout, query.Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to must be a YYYY-MM-DD day: %w", err)
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, errors.New("to must not be before from")
	}
	return from, to, nil
}

func validateGranularity(granularity string) error {
	switch granularity {
	case granularityDay, granularityWeek, granularityMonth:
		return nil
	}
	return errors.New("granularity must be day, week or month")
}

func (h *Handler) handleTimeseries(db *sql.DB, cache *dayCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		granularity := query.Get("granularity")
		err := validateGranularity(granularity)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		from, to, err := parseDayRange(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// checked first so countPeriods never loops over an unbounded range
		if rangeDays(from, to) > maxTimeseriesDays || countPeriods(from, to, granularity) > maxTimeseriesPoints {
			http.Error(w, fmt.Sprintf("range has more than %d points, use a coarser granularity", maxTimeseriesPoints), http.StatusBadRequest)
			return
		}

		response, err := getTimeseries(r.Context(), db, cache, time.Now(), from, to, granularity)
		if err != nil {
			log.Printf("error getting timeseries: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (h *Handler) handleWorstTrips(db *sql.DB, cache *dayCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		from, to, err := parseDayRange(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if rangeDays(from, to) > maxWorstTripsRangeDays {
			http.Error(w, fmt.Sprintf("range is longer than %d days", maxWorstTripsRangeDays), http.StatusBadRequest)
			return
		}

		serviceType := query.Get("service_type")
		worstTrips, err := getWorstTrips(r.Context(), db, cache, time.Now(), from, to, serviceType)
		if err != nil {
			log.Printf("error getting worst trips: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(worstTrips)
	}
}
