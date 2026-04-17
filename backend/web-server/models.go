package main

type ServiceStats struct {
	ServiceType    string  `json:"service_type"`
	TotalTrips     int     `json:"total_trips"`
	AvgDelay       float64 `json:"avg_delay"`
	OnTimeCount    int     `json:"on_time_count"`
	DelayedCount   int     `json:"delayed_count"`
	CancelledCount int     `json:"cancelled_count"`
}

type DashboardResponse struct {
	// Summary   ServiceStats   `json:"summary"`
	Breakdown []ServiceStats `json:"breakdown"`
}

type Trip struct {
	Id                 string `json:"id"`
	TrainNumber        string `json:"train_number"`
	ServiceType        string `json:"service_type"`
	OriginStation      string `json:"origin_station"`
	DestinationStation string `json:"destination"`

	ScheduledDeparture *string `json:"scheduled_departure"`
	ScheduledArrival   *string `json:"scheduled_arrival"`
	ActualDeparture    *string `json:"actual_departure"`
	ActualArrival      *string `json:"actual_arrival"`

	DelayMinutes *int   `json:"delay_minutes"`
	IsCancelled  *bool  `json:"is_cancelled"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type LeaderboardEntry struct {
	TrainNumber        string  `json:"train_number"`
	ServiceType        string  `json:"service_type"`
	OriginStation      string  `json:"origin_station"`
	DestinationStation string  `json:"destination"`
	Value              float64 `json:"value"` // avg delay, % of cancelled, etc
	Count              int     `json:"count"` // number of trips considered for the Value
}
