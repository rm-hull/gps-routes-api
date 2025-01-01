package services

import (
	"context"
	"time"

	openapi "github.com/rm-hull/gps-routes-api/go"
	repo "github.com/rm-hull/gps-routes-api/repositories"
)

type RoutesService interface {
	GetRouteByID(objectID string) (*openapi.RouteMetadata, error)
	Search(criteria *openapi.SearchRequest) (*openapi.SearchResults, error)
}

type RoutesServiceImpl struct {
	Repository repo.RouteRepository
}

func (service *RoutesServiceImpl) GetRouteByID(objectID string) (*openapi.RouteMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return service.Repository.FindByObjectID(ctx, objectID)
}

func (service *RoutesServiceImpl) Search(criteria *openapi.SearchRequest) (*openapi.SearchResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Second)
	defer cancel()

	type Facet struct {
		Name   string
		Values map[string]int64
	}

	totalChan := make(chan int64, 1)
	resultsChan := make(chan []openapi.RouteSummary, 1)
	facetsChan := make(chan Facet, 4)
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

	fetchFacet := func(fieldName string, limit int) {
		results, err := service.Repository.FacetCounts(ctx, criteria, fieldName, limit)
		if err != nil {
			errorChan <- err
		}

		facetsChan <- Facet{Name: fieldName, Values: *results}
	}

	go fetchResults()
	go fetchCounts()
	go fetchFacet("district", 20)
	go fetchFacet("county", 100)
	go fetchFacet("region", 20)
	go fetchFacet("country", 10)

	var total int64
	var results []openapi.RouteSummary

	facets := make(map[string]map[string]int64, 0)
	remaining := 6

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

	return &openapi.SearchResults{
		Hits:   results,
		Total:  total,
		Facets: facets,
	}, nil
}

func NewRoutesService(repo repo.RouteRepository) *RoutesServiceImpl {
	return &RoutesServiceImpl{Repository: repo}
}
