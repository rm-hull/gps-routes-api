package services

import (
	"context"
	"time"

	model "github.com/rm-hull/gps-routes-api/go"
	repo "github.com/rm-hull/gps-routes-api/repositories"
)

type RoutesService interface {
	GetRouteByID(objectID string) (*model.RouteMetadata, error)
	Search(criteria *model.SearchRequest) (*model.SearchResults, error)
}

type RoutesServiceImpl struct {
	Repository repo.DbRepository
}

func (service *RoutesServiceImpl) GetRouteByID(objectID string) (*model.RouteMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return service.Repository.FindByObjectID(ctx, objectID)
}

func (service *RoutesServiceImpl) Search(criteria *model.SearchRequest) (*model.SearchResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
	defer cancel()

	type Facet struct {
		Name   string
		Values map[string]int64
	}

	totalChan := make(chan int64, 1)
	resultsChan := make(chan []model.RouteSummary, 1)
	facetsChan := make(chan Facet, 5)
	errorChan := make(chan error, 2)

	fetchCounts := func() {
		total, err := service.Repository.CountAll(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		totalChan <- total
	}

	fetchResults := func() {
		results, err := service.Repository.SearchHits(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		resultsChan <- *results
	}

	fetchFacet := func(fieldName string, limit int32, unnest bool, excludeFacets ...string) {
		results, err := service.Repository.FacetCounts(ctx, criteria, fieldName, limit, unnest, excludeFacets...)
		if err != nil {
			errorChan <- err
		}

		facetsChan <- Facet{Name: fieldName, Values: *results}
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
	var results []model.RouteSummary

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

	return &model.SearchResults{
		Hits:   results,
		Total:  total,
		Facets: facets,
		Attribution: []string{
			"GPS Cycle and Walking Routes: https://gps-routes.co.uk",
			"OS DataHub: Names API. https://www.ordnancesurvey.co.uk",
			"Nominatim: Data © OpenStreetMap contributors, ODbL 1.0. http://osm.org/copyright",
		},
	}, nil
}

func NewRoutesService(repo repo.DbRepository) *RoutesServiceImpl {
	return &RoutesServiceImpl{Repository: repo}
}
