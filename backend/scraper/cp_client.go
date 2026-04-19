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
	Delay            *int             `json:"delay"`      // The delay in minutes. can be null, can be 0 or a positive number
	Supression       *Supression      `json:"supression"` // null if not cancelled
	ETA              *string          `json:"ETA"`        // can be null - typically ArrivalTime + delay, but not always - in those cases not clear if delay or this has the real delay
	ETD              *string          `json:"ETD"`        // can be null
}

type TrainServiceInfo struct {
	Code        string `json:"code"`        // e.g. "IC"
	Designation string `json:"designation"` // e.g. "Intercidades"
}

type StationInfo struct {
	Code        string `json:"code"`
	Designation string `json:"designation"`
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
			Timeout: 15 * time.Second,
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

func (c *CPClient) FetchStations(_ context.Context) ([]StationInfo, error) {
	// TODO: use all relevant stations from https://api-gateway.cp.pt/cp/services/travel-api/stations
	return []StationInfo{

		// algarve
		{Designation: "Faro", Code: "94-73007"},
		{Designation: "Lagos", Code: "94-90464"},
		{Designation: "Vila Real de Santo António", Code: "94-73569"},

		// alentejo
		{Designation: "Évora", Code: "94-83006"},
		{Designation: "Beja", Code: "94-75002"},
		{Designation: "Casa Branca", Code: "94-74005"},
		{Designation: "Badajoz", Code: "11-37606"},

		// lisboa AML
		{Designation: "Lisboa Oriente", Code: "94-31039"},
		{Designation: "Lisboa Santa Apolónia", Code: "94-30007"},
		{Designation: "Lisboa Rossio", Code: "94-59006"},
		{Designation: "Alcantara Terra", Code: "94-67025"},
		{Designation: "Alcantara Mar", Code: "94-69039"},
		{Designation: "Setúbal", Code: "94-68122"},
		{Designation: "Praias do Sado", Code: "94-91058"},
		{Designation: "Sintra", Code: "94-61101"},
		{Designation: "Cascais", Code: "94-69260"},
		{Designation: "Cais Sodre", Code: "94-69005"},
		{Designation: "Azambuja", Code: "94-33001"},
		{Designation: "Castanheira do Ribatejo", Code: "94-31310"},
		{Designation: "Meleças", Code: "94-62042"},
		{Designation: "Barreiro", Code: "94-95000"},

		// center
		{Designation: "Entroncamento", Code: "94-34009"},
		{Designation: "Abrantes", Code: "94-52001"},
		{Designation: "Coimbra-B", Code: "94-36004"},
		{Designation: "Tomar", Code: "94-40154"},

		// west
		{Designation: "Caldas da Rainha", Code: "94-63008"},
		{Designation: "Figueira da Foz", Code: "94-64113"},
		{Designation: "Leiria", Code: "94-63560"},

		// Beiras
		{Designation: "Vilar Formoso", Code: "94-49460"},
		{Designation: "Castelo Branco", Code: "94-53009"},
		{Designation: "Guarda", Code: "94-49007"},
		{Designation: "Covilhã", Code: "94-54007"},

		// --- NORTH (Porto Urbanos, Minho, Douro) ---
		{Designation: "Porto Campanhã", Code: "94-2006"},
		{Designation: "Porto São Bento", Code: "94-1008"},
		{Designation: "Valenca", Code: "94-7005"},
		{Designation: "Braga", Code: "94-29157"},
		{Designation: "Guimarães", Code: "94-24000"},
		{Designation: "Aveiro", Code: "94-38000"},
		{Designation: "Viana do Castelo", Code: "94-18002"},
		{Designation: "Pocinho", Code: "94-12005"},
		{Designation: "Régua", Code: "94-10009"},
		{Designation: "Marco Canaveses", Code: "94-9001"},
		{Designation: "Nine", Code: "94-6007"},
		{Designation: "Vigo-Guixar", Code: "11-22308"},
		{Designation: "Ovar", Code: "11-38299"},
		{Designation: "Granja", Code: "11-39040"},
		{Designation: "Espinho Vouga", Code: "11-44016"},
		{Designation: "Oliveira Azemeis", Code: "11-44339"},
		{Designation: "Sernada Vouga", Code: "11-43000"},
		{Designation: "Agueda", Code: "11-42218"},
	}, nil

}
