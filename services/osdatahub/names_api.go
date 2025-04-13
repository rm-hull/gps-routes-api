package osdatahub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Depado/ginprom"
)

type Payload struct {
	Header  Header   `json:"header"`
	Results []Result `json:"results"`
}

type Header struct {
	URI          string `json:"uri"`
	Query        string `json:"query"`
	Format       string `json:"format"`
	MaxResults   int32  `json:"maxresults"`
	Offset       int32  `json:"offset"`
	TotalResults int32  `json:"totalresults"`
}

type ErrorPayload struct {
	Fault Fault `json:"fault"`
}

type Fault struct {
	FaultString string `json:"faultstring"`
	Detail      struct {
		ErrorCode string `json:"errorcode"`
	} `json:"detail"`
}

type Result struct {
	GazetteerEntry GazetteerEntry `json:"GAZETTEER_ENTRY"`
}

type GazetteerEntry struct {
	ID                  string  `json:"ID"`
	NamesURI            string  `json:"NAMES_URI"`
	Name1               string  `json:"NAME1"`
	Type                string  `json:"TYPE"`
	LocalType           string  `json:"LOCAL_TYPE"`
	GeometryX           float64 `json:"GEOMETRY_X"`
	GeometryY           float64 `json:"GEOMETRY_Y"`
	MostDetailViewRes   int32   `json:"MOST_DETAIL_VIEW_RES"`
	LeastDetailViewRes  int32   `json:"LEAST_DETAIL_VIEW_RES"`
	MbrXmin             float64 `json:"MBR_XMIN"`
	MbrYmin             float64 `json:"MBR_YMIN"`
	MbrXmax             float64 `json:"MBR_XMAX"`
	MbrYmax             float64 `json:"MBR_YMAX"`
	PostcodeDistrict    string  `json:"POSTCODE_DISTRICT"`
	PostcodeDistrictURI string  `json:"POSTCODE_DISTRICT_URI"`
	PopulatedPlace      string  `json:"POPULATED_PLACE"`
	PopulatedPlaceURI   string  `json:"POPULATED_PLACE_URI"`
	PopulatedPlaceType  string  `json:"POPULATED_PLACE_TYPE"`
	CountyUnitary       string  `json:"COUNTY_UNITARY"`
	CountyUnitaryURI    string  `json:"COUNTY_UNITARY_URI"`
	CountyUnitaryType   string  `json:"COUNTY_UNITARY_TYPE"`
	Region              string  `json:"REGION"`
	RegionURI           string  `json:"REGION_URI"`
	Country             string  `json:"COUNTRY"`
	CountryURI          string  `json:"COUNTRY_URI"`
}

type NamesApi interface {
	Find(name string) (*Result, error)
}

type NamesApiImpl struct {
	baseUrl    string
	apiKey     string
	prometheus *ginprom.Prometheus
}

func NewNamesApi(prometheus *ginprom.Prometheus, baseUrl string, apiKey string) *NamesApiImpl {
	if prometheus != nil {
		prometheus.AddCustomCounter("os_names_api_call_count", "Number of calls to OS Names API", []string{})
	}

	return &NamesApiImpl{
		baseUrl:    baseUrl,
		apiKey:     apiKey,
		prometheus: prometheus,
	}
}

func (service *NamesApiImpl) Find(name string) (*Result, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", service.baseUrl+"/find", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("maxresults", "1")
	q.Add("format", "JSON")
	q.Add("query", name)
	q.Add("key", service.apiKey)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != 200 {
		var errorPayload ErrorPayload
		err = json.Unmarshal(body, &errorPayload)
		if err != nil {
			return nil, fmt.Errorf("error parsing fault JSON: %w", err)
		}

		return nil, fmt.Errorf("bad response (%s): %s", resp.Status, errorPayload.Fault.FaultString)
	}

	var payload Payload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	if service.prometheus != nil {
		service.prometheus.IncrementCounterValue("os_names_api_call_count", []string{}) //nolint:errcheck
	}

	if payload.Header.TotalResults == 0 {
		return nil, nil
	}
	latLng := ToWSG84(&payload.Results[0])
	fmt.Printf("latLng: %v", latLng)

	return &payload.Results[0], nil
}
