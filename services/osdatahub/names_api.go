package osdatahub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Depado/ginprom"
	"github.com/kofalt/go-memoize"
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
	Find(ctx context.Context, name string) (*Result, error)
}

type NamesApiImpl struct {
	baseUrl    string
	apiKey     string
	prometheus *ginprom.Prometheus
	cache      *memoize.Memoizer
}

func NewNamesApi(prometheus *ginprom.Prometheus, baseUrl string, apiKey string) *NamesApiImpl {
	if prometheus != nil {
		prometheus.AddCustomCounter("names_api_cache_stats_total", "Number of calls to OS Names API cache statistics (hits & misses)", []string{"type"})
		prometheus.AddCustomGauge("names_api_cache_size", "OS Names cache size (number of items)", []string{})
	}

	return &NamesApiImpl{
		baseUrl:    baseUrl,
		apiKey:     apiKey,
		prometheus: prometheus,
		cache:      memoize.NewMemoizer(8*time.Hour, 24*time.Hour),
	}
}

func (service *NamesApiImpl) Find(ctx context.Context, name string) (*Result, error) {
	result, err, cached := memoize.Call(service.cache, name, func() (*Result, error) {
		return service.fetch(ctx, name)
	})
	service.updateMetrics(cached)
	return result, err
}

func (service *NamesApiImpl) updateMetrics(cached bool) {
	if service.prometheus != nil {
		service.prometheus.SetGaugeValue("names_api_cache_size", []string{}, float64(service.cache.Storage.ItemCount())) //nolint:errcheck
		if cached {
			service.prometheus.IncrementCounterValue("names_api_cache_stats_total", []string{"hit"}) //nolint:errcheck
		} else {
			service.prometheus.IncrementCounterValue("names_api_cache_stats_total", []string{"miss"}) //nolint:errcheck
		}
	}
}

func (service *NamesApiImpl) fetch(ctx context.Context, name string) (*Result, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", service.baseUrl+"/find", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("maxresults", "1")
	q.Add("format", "JSON")
	q.Add("query", name)
	q.Add("key", service.apiKey)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
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

	if payload.Header.TotalResults == 0 {
		return nil, nil
	}
	latLng := ToWSG84(&payload.Results[0])
	fmt.Printf("latLng: %v", latLng)

	return &payload.Results[0], nil
}
