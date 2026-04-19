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

// stations is a map of station codes to StationInfo
var stations = map[string]StationInfo{
	// algarve
	"94-73007": {Designation: "Faro", Code: "94-73007"},
	"94-90464": {Designation: "Lagos", Code: "94-90464"},
	"94-73569": {Designation: "Vila Real de Santo António", Code: "94-73569"},

	// alentejo
	"94-83006": {Designation: "Évora", Code: "94-83006"},
	"94-75002": {Designation: "Beja", Code: "94-75002"},
	"94-74005": {Designation: "Casa Branca", Code: "94-74005"},
	"11-37606": {Designation: "Badajoz", Code: "11-37606"},

	// lisboa AML
	"94-31039": {Designation: "Lisboa Oriente", Code: "94-31039"},
	"94-30007": {Designation: "Lisboa Santa Apolónia", Code: "94-30007"},
	"94-59006": {Designation: "Lisboa Rossio", Code: "94-59006"},
	"94-67025": {Designation: "Alcantara Terra", Code: "94-67025"},
	"94-69039": {Designation: "Alcantara Mar", Code: "94-69039"},
	"94-68122": {Designation: "Setúbal", Code: "94-68122"},
	"94-91058": {Designation: "Praias do Sado", Code: "94-91058"},
	"94-61101": {Designation: "Sintra", Code: "94-61101"},
	"94-69260": {Designation: "Cascais", Code: "94-69260"},
	"94-69005": {Designation: "Cais Sodre", Code: "94-69005"},
	"94-33001": {Designation: "Azambuja", Code: "94-33001"},
	"94-31310": {Designation: "Castanheira do Ribatejo", Code: "94-31310"},
	"94-62042": {Designation: "Meleças", Code: "94-62042"},
	"94-95000": {Designation: "Barreiro", Code: "94-95000"},

	// center
	"94-34009": {Designation: "Entroncamento", Code: "94-34009"},
	"94-52001": {Designation: "Abrantes", Code: "94-52001"},
	"94-36004": {Designation: "Coimbra-B", Code: "94-36004"},
	"94-40154": {Designation: "Tomar", Code: "94-40154"},

	// west
	"94-63008": {Designation: "Caldas da Rainha", Code: "94-63008"},
	"94-64113": {Designation: "Figueira da Foz", Code: "94-64113"},
	"94-63560": {Designation: "Leiria", Code: "94-63560"},

	// Beiras
	"94-49460": {Designation: "Vilar Formoso", Code: "94-49460"},
	"94-53009": {Designation: "Castelo Branco", Code: "94-53009"},
	"94-49007": {Designation: "Guarda", Code: "94-49007"},
	"94-54007": {Designation: "Covilhã", Code: "94-54007"},

	// --- NORTH (Porto Urbanos, Minho, Douro) ---
	"94-2006":  {Designation: "Porto Campanhã", Code: "94-2006"},
	"94-1008":  {Designation: "Porto São Bento", Code: "94-1008"},
	"94-7005":  {Designation: "Valenca", Code: "94-7005"},
	"94-29157": {Designation: "Braga", Code: "94-29157"},
	"94-24000": {Designation: "Guimarães", Code: "94-24000"},
	"94-38000": {Designation: "Aveiro", Code: "94-38000"},
	"94-18002": {Designation: "Viana do Castelo", Code: "94-18002"},
	"94-12005": {Designation: "Pocinho", Code: "94-12005"},
	"94-10009": {Designation: "Régua", Code: "94-10009"},
	"94-9001":  {Designation: "Marco Canaveses", Code: "94-9001"},
	"94-6007":  {Designation: "Nine", Code: "94-6007"},
	"11-22308": {Designation: "Vigo-Guixar", Code: "11-22308"},
	"11-38299": {Designation: "Ovar", Code: "11-38299"},
	"11-39040": {Designation: "Granja", Code: "11-39040"},
	"11-44016": {Designation: "Espinho Vouga", Code: "11-44016"},
	"11-44339": {Designation: "Oliveira Azemeis", Code: "11-44339"},
	"11-43000": {Designation: "Sernada Vouga", Code: "11-43000"},
	"11-42218": {Designation: "Agueda", Code: "11-42218"},
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
