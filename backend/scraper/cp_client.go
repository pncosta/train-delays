package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
	"train-delays/shared"
)

type Trip struct {
	TrainNumber      int                `json:"trainNumber"`
	TrainService     TrainServiceInfo   `json:"trainService"`
	TrainOrigin      shared.StationInfo `json:"trainOrigin"`
	TrainDestination shared.StationInfo `json:"trainDestination"`
	ArrivalTime      *string            `json:"arrivalTime"`   // Can be null
	DepartureTime    *string            `json:"departureTime"` // Can be null
	Delay            *int               `json:"delay"`         // The delay in minutes. can be null, can be 0 or a positive number
	Supression       *Supression        `json:"supression"`    // null if not cancelled
	ETA              *string            `json:"ETA"`           // can be null - typically ArrivalTime + delay, but not always - in those cases not clear if delay or this has the real delay
	ETD              *string            `json:"ETD"`           // can be null
}

type TrainServiceInfo struct {
	Code        string `json:"code"`        // e.g. "IC"
	Designation string `json:"designation"` // e.g. "Intercidades"
}

type Supression struct {
	Code        string `json:"code"`
	Designation string `json:"designation"`
}

// TrainListEntry is one entry of GET /cp/services/travel-api/trains. We only use the
// train number - service/origin/destination come from the per-train timetable instead.
type TrainListEntry struct {
	TrainNumber int `json:"trainNumber"`
}

// TrainTimetable is the response of GET /cp/services/travel-api/trains/{trainNumber}/timetable/{date}.
// Only origin/destination stops + top-level delay are captured - platform, lat/long,
// occupancy, live per-stop delay, and intermediate stops are out of scope.
type TrainTimetable struct {
	ServiceCode TrainServiceInfo `json:"serviceCode"`
	Status      *string          `json:"status"` // observed: null, AT_ORIGIN, IN_TRANSIT, AT_STATION, COMPLETED
	Delay       *int             `json:"delay"`
	TrainStops  []TrainStop      `json:"trainStops"`
}

type TrainStop struct {
	Station    shared.StationInfo `json:"station"`
	Arrival    *string            `json:"arrival"`
	Departure  *string            `json:"departure"`
	ETA        *string            `json:"ETA"`
	ETD        *string            `json:"ETD"`
	Supression *Supression        `json:"supression"`
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
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &CPClient{
		BaseURL:       baseURL,
		ApiKey:        apiKey,
		ConnectID:     connectID,
		ConnectSecret: connectSecret,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

// newRequest builds a GET request against the CP API with the headers every endpoint needs.
func (c *CPClient) newRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("x-cp-connect-id", c.ConnectID)
	req.Header.Set("x-cp-connect-secret", c.ConnectSecret)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.cp.pt/")
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	return req, nil
}

// apiStatusError carries the HTTP status and a body excerpt for a non-200 CP response, so
// callers can distinguish specific failure conditions (e.g. the calendar-invalid-date 500)
// from other errors instead of treating every failure the same way.
type apiStatusError struct {
	StatusCode int
	Body       string
}

func (e *apiStatusError) Error() string {
	return fmt.Sprintf("api returned status %d: %s", e.StatusCode, e.Body)
}

// isCalendarInvalidDate reports whether err is CP's signal that the requested date isn't
// valid for this train (observed as HTTP 500 - confirmed via live testing against the
// per-train timetable endpoint), as opposed to a different/transient failure.
func isCalendarInvalidDate(err error) bool {
	var apiErr *apiStatusError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusInternalServerError
}

func (c *CPClient) do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		return nil, &apiStatusError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return resp, nil
}

// FetchTrains fetches the static list of all train numbers CP runs.
func (c *CPClient) FetchTrains(ctx context.Context) ([]TrainListEntry, error) {
	url := fmt.Sprintf("%s/cp/services/travel-api/trains", c.BaseURL)

	req, err := c.newRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching train list: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	var result []TrainListEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode error: %w", err)
	}

	return result, nil
}

// FetchTrainTimetable fetches a single train's timetable for the given date. date is a
// calendar-validity gate (CP 500s for a date the train doesn't run), not a data selector -
// once valid, the response reflects the train's current/most recent run regardless of
// which valid date was passed.
func (c *CPClient) FetchTrainTimetable(ctx context.Context, trainNumber int, date string) (*TrainTimetable, error) {
	url := fmt.Sprintf("%s/cp/services/travel-api/trains/%d/timetable/%s", c.BaseURL, trainNumber, date)

	req, err := c.newRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching timetable for train %d: %w", trainNumber, err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	var result TrainTimetable
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode error: %w", err)
	}

	return &result, nil
}
