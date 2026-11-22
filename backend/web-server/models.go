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
	Id          string `json:"id"`
	TrainNumber string `json:"train_number"`
	ServiceType string `json:"service_type"`

	OriginStation          string `json:"origin_station"`
	OriginStationName      string `json:"origin_station_name"`
	DestinationStation     string `json:"destination_station"`
	DestinationStationName string `json:"destination_station_name"`

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
	TrainNumber            string  `json:"train_number"`
	ServiceType            string  `json:"service_type"`
	OriginStation          string  `json:"origin_station"`
	OriginStationName      string  `json:"origin_station_name"`
	DestinationStation     string  `json:"destination_station"`
	DestinationStationName string  `json:"destination_station_name"`
	Value                  float64 `json:"value"` // avg delay, % of cancelled, etc
	Count                  int     `json:"count"` // number of trips considered for the Value
}

// DailyStats mirrors a daily_stats row; the same type accumulates a period's totals.
type DailyStats struct {
	Day         string
	ServiceType string

	Trips     int
	Cancelled int
	OnTime    int
	Delayed   int

	DelaySum   int
	DelayCount int
	MaxDelay   int

	LateUpTo10 int
	Late10To30 int
	LateOver30 int
}

// DailyWorstTrip mirrors a daily_worst_trips row; NULL columns come back as zero values.
type DailyWorstTrip struct {
	Day                    string `json:"day"`
	ServiceType            string `json:"service_type"`
	TripID                 string `json:"trip_id"`
	TrainNumber            int    `json:"train_number"`
	OriginStation          string `json:"origin_station"`
	OriginStationName      string `json:"origin_station_name"`
	DestinationStation     string `json:"destination_station"`
	DestinationStationName string `json:"destination_station_name"`
	ScheduledArrival       string `json:"scheduled_arrival"`
	ActualArrival          string `json:"actual_arrival"`
	DelayMinutes           int    `json:"delay_minutes"`
}

type TimeseriesStats struct {
	ServiceType string  `json:"service_type"`
	Trips       int     `json:"trips"`
	Cancelled   int     `json:"cancelled"`
	OnTime      int     `json:"on_time"`
	Delayed     int     `json:"delayed"`
	AvgDelay    float64 `json:"avg_delay"`
	LateUpTo10  int     `json:"late_up_to_10"`
	Late10To30  int     `json:"late_10_30"`
	LateOver30  int     `json:"late_over_30"`
	MaxDelay    int     `json:"max_delay"`
}

type TimeseriesPeriod struct {
	Period string            `json:"period"`
	Stats  []TimeseriesStats `json:"stats"`
}

type TimeseriesResponse struct {
	Granularity string             `json:"granularity"`
	Periods     []TimeseriesPeriod `json:"periods"`
}

// summaryGroup is a count of trips sharing the same values of the columns /summary classifies on.
type summaryGroup struct {
	ServiceType  string
	DelayMinutes *int
	IsCancelled  bool
	ArrivalSeen  bool
	Count        int
}
