package main

type SummaryResponse struct {
	TotalTrains    int     `json:"total_trains"`
	TotalCancelled int     `json:"total_cancelled"`
	TotalDelayed   int     `json:"total_delayed"`
	AvgDelay       float64 `json:"avg_delay"`
}
