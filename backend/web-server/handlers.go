package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
