package repositories

import (
	"context"

	model "github.com/rm-hull/gps-routes-api/go"
)

type DbRepository interface {
	Store(ctx context.Context, route *model.RouteMetadata) error
	FindByObjectID(ctx context.Context, objectID string) (*model.RouteMetadata, error)
	CountAll(ctx context.Context, criteria *model.SearchRequest) (int64, error)
	SearchHits(ctx context.Context, criteria *model.SearchRequest) (*[]model.RouteSummary, error)
	FacetCounts(ctx context.Context, criteria *model.SearchRequest, facetField string, limit int) (*map[string]int64, error)
}
