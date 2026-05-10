package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"train-delays/shared"
)

type TimetableResponse struct {
	Trips []Trip `json:"stationStops"`
}

type Trip struct {
	TrainNumber      int                `json:"trainNumber"`
	TrainService     TrainServiceInfo   `json:"trainService"`
	TrainOrigin      shared.StationInfo `json:"trainOrigin"`
	TrainDestination shared.StationInfo `json:"trainDestination"`
	ArrivalTime      *string            `json:"arrivalTime"`   // Can be null
	DepartureTime    *string            `json:"departureTime"` // Can be null
	Platform         string             `json:"platform"`
	Delay            *int               `json:"delay"`      // The delay in minutes. can be null, can be 0 or a positive number
	Supression       *Supression        `json:"supression"` // null if not cancelled
	ETA              *string            `json:"ETA"`        // can be null - typically ArrivalTime + delay, but not always - in those cases not clear if delay or this has the real delay
	ETD              *string            `json:"ETD"`        // can be null
}

type TrainServiceInfo struct {
	Code        string `json:"code"`        // e.g. "IC"
	Designation string `json:"designation"` // e.g. "Intercidades"
}

type Supression struct {
	Code        string `json:"code"`
	Designation string `json:"designation"`
}

// Client handles communication with the CP API
type CPClient struct {
	BaseURL       string
	ApiKey        string
	ConnectID     string
	ConnectSecret string
	HTTPClient    *http.Client
}

// NewCPClient initializes the CP client
func NewCPClient(baseURL, apiKey, connectID, connectSecret string) *CPClient {
	return &CPClient{
		BaseURL:       baseURL,
		ApiKey:        apiKey,
		ConnectID:     connectID,
		ConnectSecret: connectSecret,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchTimetable performs the HTTP GET and decodes the response
func (c *CPClient) FetchTrips(ctx context.Context, stationID string, startTime time.Time) ([]Trip, error) {

	date := startTime.Format("2006-01-02") // this seems to be mostly ignored by the API.. but still needed
	startHour := startTime.Format("15:04")
	endpoint := "cp/services/travel-api/stations"
	url := fmt.Sprintf("%s/%s/%s/timetable/%s?start=%s", c.BaseURL, endpoint, stationID, date, startHour)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("x-cp-connect-id", c.ConnectID)
	req.Header.Set("x-cp-connect-secret", c.ConnectSecret)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", resp.StatusCode)
	}

	var result TimetableResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode error: %w", err)
	}

	return result.Trips, nil
}
