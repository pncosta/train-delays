package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TimetableResponse struct {
	Trips []Trip `json:"stationStops"`
}

type Trip struct {
	TrainNumber      int              `json:"trainNumber"`
	TrainService     TrainServiceInfo `json:"trainService"`
	TrainOrigin      StationInfo      `json:"trainOrigin"`
	TrainDestination StationInfo      `json:"trainDestination"`
	ArrivalTime      *string          `json:"arrivalTime"`   // Can be null
	DepartureTime    *string          `json:"departureTime"` // Can be null
	Platform         string           `json:"platform"`
	Delay            *int             `json:"delay"` // The integer delay you saw
	// Supression    *string `json:"supression"` // Can be null
	ETA *string `json:"ETA"` // Estimated Time of Arrival
	ETD *string `json:"ETD"` // Estimated Time of Departure
}

type TrainServiceInfo struct {
	Code        string `json:"code"`        // e.g. "IC"
	Designation string `json:"designation"` // e.g. "Intercidades"
}

type StationInfo struct {
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
			Timeout: 15 * time.Second,
		},
	}
}

// FetchTimetable performs the HTTP GET and decodes the response
func (c *CPClient) FetchTimetable(ctx context.Context, stationID string, date string) (*TimetableResponse, error) {
	// Format: .../94-31039/timetable/2026-03-23?start=00:00
	url := fmt.Sprintf("%s/%s/timetable/%s?start=00:00", c.BaseURL, stationID, date)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("x-cp-connect-id", c.ConnectID)
	req.Header.Set("x-cp-connect-secret", c.ConnectSecret)

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

	return &result, nil
}

func (c *CPClient) FetchStations(ctx context.Context) ([]StationInfo, error) {
	// TODO: use all relevant stations from https://api-gateway.cp.pt/cp/services/travel-api/stations
	lisboaOriente := StationInfo{
		Designation: "Lisboa Oriente",
		Code:        "94-31039",
	}
	lisboaSA := StationInfo{
		Designation: "Lisboa SA",
		Code:        "94-30007",
	}
	portoCampanha := StationInfo{
		Designation: "Porto Campanha",
		Code:        "94-2006",
	}
	amadora := StationInfo{
		Designation: "Amadora",
		Code:        "94-60087",
	}

	return []StationInfo{lisboaSA, lisboaOriente, portoCampanha, amadora}, nil
}
