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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	totalChan := make(chan int64, 1)
	resultsChan := make(chan []openapi.RouteSummary, 1)
	errorChan := make(chan error, 2)

	go func() {
		total, err := service.Repository.CountAll(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		totalChan <- total
	}()

	go func() {
		results, err := service.Repository.SearchHits(ctx, criteria)
		if err != nil {
			errorChan <- err
			return
		}
		resultsChan <- *results
	}()

	var total int64
	var results []openapi.RouteSummary
	remaining := 2

	for remaining > 0 {
		select {
		case err := <-errorChan:
			return nil, err
		case total = <-totalChan:
			remaining--
		case results = <-resultsChan:
			remaining--
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return &openapi.SearchResults{
		Hits:  results,
		Total: int32(total),
	}, nil
}

func NewRoutesService(repo repo.RouteRepository) *RoutesServiceImpl {
	return &RoutesServiceImpl{Repository: repo}
}
