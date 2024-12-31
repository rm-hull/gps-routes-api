package services

import (
	"context"
	"time"

	model "github.com/rm-hull/gps-routes-api/go"
	repo "github.com/rm-hull/gps-routes-api/repositories"
)

type RoutesService interface {
	GetRouteByID(objectID string) (*model.RouteMetadata, error)
}

type RoutesServiceImpl struct {
	Repository repo.RouteRepository
}

func (service *RoutesServiceImpl) GetRouteByID(objectID string) (*model.RouteMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return service.Repository.FindByObjectID(ctx, objectID)
}

func NewRoutesService(repo repo.RouteRepository) *RoutesServiceImpl {
	return &RoutesServiceImpl{Repository: repo}
}
