package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rm-hull/gps-routes-api/models/domain"
	"github.com/rm-hull/gps-routes-api/models/request"
	repo "github.com/rm-hull/gps-routes-api/repositories"
	"github.com/rm-hull/gps-routes-api/services/osdatahub"
)

type RoutesService interface {
	GetRouteByID(objectID string) (*domain.RouteMetadata, error)
	Search(criteria *request.SearchRequest) (*domain.SearchResults, error)
}

type RoutesServiceImpl struct {
	repository    repo.DbRepository
	namesApi      osdatahub.NamesApi
	searchTimeout time.Duration
}

func (service *RoutesServiceImpl) GetRouteByID(objectID string) (*domain.RouteMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return service.repository.FindByObjectID(ctx, objectID)
}

type facet struct {
	Name   string
	Values map[string]int64
}

func (service *RoutesServiceImpl) Search(criteria *request.SearchRequest) (*domain.SearchResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), service.searchTimeout)
	defer cancel()

	if criteria.Nearby != nil && criteria.Nearby.Place != "" && criteria.Nearby.Center == nil {
		result, err := service.namesApi.Find(ctx, criteria.Nearby.Place)
		if err != nil {
			return nil, fmt.Errorf("error fetching from osdatahub Names API: %w", err)
		}
		criteria.Nearby.Center = osdatahub.ToWSG84(result)
	}

	totalChan := make(chan int64, 1)
	resultsChan := make(chan []domain.RouteSummary, 1)
	facetsChan := make(chan facet, 5)
	errorChan := make(chan error, 2)

	fetchCounts := func() {
		total, err := service.repository.CountAll(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		totalChan <- total
	}

	fetchResults := func() {
		results, err := service.repository.SearchHits(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		resultsChan <- *results
	}

	fetchFacet := func(fieldName string, limit int32, unnest bool, excludeFacets ...string) {
		results, err := service.repository.FacetCounts(ctx, criteria, fieldName, limit, unnest, excludeFacets...)
		if err != nil {
			errorChan <- err
		}

		facetsChan <- facet{Name: fieldName, Values: *results}
	}

	go fetchResults()
	go fetchCounts()
	go fetchFacet("district", 20, false, "district")
	go fetchFacet("county", 100, false, "county", "district")
	go fetchFacet("region", 20, false, "region", "county", "district")
	go fetchFacet("state", 20, false, "state", "region", "county", "district")
	go fetchFacet("country", 10, false, "country", "state", "region", "county", "district")
	go fetchFacet("route_type", 10, false, "route_type")
	go fetchFacet("difficulty", 10, false, "difficulty")
	go fetchFacet("estimated_duration", 10, false, "estimated_duration")
	go fetchFacet("terrain", 50, true, "terrain")
	go fetchFacet("facilities", 50, true, "facilities")
	go fetchFacet("activities", 50, true, "activities")
	go fetchFacet("points_of_interest", 50, true, "points_of_interest")

	var total int64
	var results []domain.RouteSummary

	facets := make(map[string]map[string]int64, 0)
	remaining := 14

	for remaining > 0 {
		select {
		case err := <-errorChan:
			return nil, err
		case total = <-totalChan:
			remaining--
		case results = <-resultsChan:
			remaining--
		case facet := <-facetsChan:
			remaining--
			facets[facet.Name] = facet.Values
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &domain.SearchResults{
		Hits:   results,
		Total:  total,
		Facets: facets,
		Attribution: []string{
			"GPS Cycle and Walking Routes: https://gps-routes.co.uk",
			"OS DataHub: Names API. https://www.ordnancesurvey.co.uk",
			"Nominatim: Data © OpenStreetMap contributors, ODbL 1.0. http://osm.org/copyright",
			"Llama.cpp: https://github.com/ggml-org/llama.cpp/blob/master/LICENSE",
			"lmstudio-community/gemma-3-4b-it-Q8_0.gguf: https://ai.google.dev/gemma/terms",
		},
	}, nil
}

func NewRoutesService(repo repo.DbRepository, namesApi osdatahub.NamesApi) *RoutesServiceImpl {
	return &RoutesServiceImpl{
		repository:    repo,
		namesApi:      namesApi,
		searchTimeout: 10 * time.Second, // TODO: Consider changing to configurable options
	}
}
