package main

import (
	"context"
	"encoding/json"
	"net/http"
)

type Handler struct {
}

func (h *Handler) handleSummary(_ context.Context, _ string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysParam := r.URL.Query().Get("days")
		if daysParam == "" {
			daysParam = "7" // Default to last week
		}

		res := &SummaryResponse{
			TotalTrains:    400,
			TotalCancelled: 2,
			TotalDelayed:   100,
			AvgDelay:       4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}
