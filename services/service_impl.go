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

var ATTRIBUTION = []string{
	"GPS Cycle and Walking Routes: https://gps-routes.co.uk",
	"OS DataHub: Names API. https://www.ordnancesurvey.co.uk",
	"Nominatim: Data © OpenStreetMap contributors, ODbL 1.0. http://osm.org/copyright",
	"Llama.cpp: https://github.com/ggml-org/llama.cpp/blob/master/LICENSE",
	"lmstudio-community/gemma-3-4b-it-Q8_0.gguf: https://ai.google.dev/gemma/terms",
}

type RoutesService interface {
	GetRouteByID(objectID string) (*domain.RouteMetadata, error)
	Search(criteria *request.SearchRequest) (*domain.SearchResults, error)
	RefData() (*domain.RefData, error)
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
	facetsChan := make(chan facet, len(FACET_FIELDS))
	errorChan := make(chan error, 2+len(FACET_FIELDS)) // 2 for total and results, plus the number of facets

	fetchCounts := func() {
		total, err := service.repository.CountAll(ctx, criteria)
		if err != nil {
			errorChan <- err
		} else {
			totalChan <- total
		}
	}

	fetchResults := func() {
		results, err := service.repository.SearchHits(ctx, criteria)
		if err != nil {
			errorChan <- err
		} else {
			resultsChan <- *results
		}
	}

	go fetchResults()
	go fetchCounts()

	var waitForFacets int

	if !criteria.SkipFacets {
		for fieldName, facetConfig := range FACET_FIELDS {
			go func() {
				results, err := service.repository.FacetCounts(ctx, criteria, fieldName, facetConfig.Limit, facetConfig.Unnest, facetConfig.Excluded...)
				if err != nil {
					errorChan <- err
				} else {
					facetsChan <- facet{Name: fieldName, Values: *results}
				}
			}()
			waitForFacets++
		}
	}

	var total int64
	var results []domain.RouteSummary

	facets := make(domain.Facets, waitForFacets)
	remaining := 2 + waitForFacets // 2 for total and results, plus the number of facets

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
		Hits:        results,
		Total:       total,
		Facets:      facets,
		Attribution: ATTRIBUTION,
	}, nil
}

func (service *RoutesServiceImpl) RefData() (*domain.RefData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), service.searchTimeout)
	defer cancel()

	facetsChan := make(chan facet, len(FACET_FIELDS))
	errorChan := make(chan error, len(FACET_FIELDS))
	emptyCriteria := request.SearchRequest{}

	for fieldName, facetConfig := range FACET_FIELDS {
		go func() {
			results, err := service.repository.FacetCounts(ctx, &emptyCriteria, fieldName, -1 /* no limit */, facetConfig.Unnest)
			if err != nil {
				errorChan <- err
			} else {
				facetsChan <- facet{Name: fieldName, Values: *results}
			}
		}()
	}

	facets := make(domain.Facets, len(FACET_FIELDS))
	remaining := len(FACET_FIELDS)

	for remaining > 0 {
		select {
		case err := <-errorChan:
			return nil, err
		case facet := <-facetsChan:
			remaining--
			facets[facet.Name] = facet.Values
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &domain.RefData{
		Facets:      facets,
		Attribution: ATTRIBUTION,
	}, nil
}

func NewRoutesService(repo repo.DbRepository, namesApi osdatahub.NamesApi) *RoutesServiceImpl {
	return &RoutesServiceImpl{
		repository:    repo,
		namesApi:      namesApi,
		searchTimeout: 10 * time.Second, // TODO: Consider changing to configurable options
	}
}
