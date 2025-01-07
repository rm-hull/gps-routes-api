package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kofalt/go-memoize"

	openapi "github.com/rm-hull/gps-routes-api/go"
)

type CachedDbRepository struct {
	cache   *memoize.Memoizer
	wrapped DbRepository
}

func NewCachedRepository(wrapped DbRepository) *CachedDbRepository {
	return &CachedDbRepository{
		cache:   memoize.NewMemoizer(90*time.Second, 10*time.Minute),
		wrapped: wrapped,
	}
}

func buildCacheKey(v interface{}) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %v", err)
	}

	hash := md5.Sum(jsonBytes)
	return fmt.Sprintf("%x", hash), nil
}

// CountAll implements DbRepository.
func (repo *CachedDbRepository) CountAll(ctx context.Context, criteria *openapi.SearchRequest) (int64, error) {
	key, err := buildCacheKey(map[string]any{
		"query":       criteria.Query,
		"boundingBox": criteria.BoundingBox,
		"facets":      criteria.Facets,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create cache key: %v", err)
	}
	result, err, _ := memoize.Call(repo.cache, key, func() (int64, error) {
		return repo.wrapped.CountAll(ctx, criteria)
	})
	return result, err
}

// FacetCounts implements DbRepository.
func (repo *CachedDbRepository) FacetCounts(ctx context.Context, criteria *openapi.SearchRequest, facetField string, limit int32) (*map[string]int64, error) {
	key, err := buildCacheKey(map[string]any{
		"query":       criteria.Query,
		"boundingBox": criteria.BoundingBox,
		"facets":      criteria.Facets,
		"field":       facetField,
		"limit":       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache key: %v", err)
	}
	result, err, _ := memoize.Call(repo.cache, key, func() (*map[string]int64, error) {
		return repo.wrapped.FacetCounts(ctx, criteria, facetField, limit)
	})
	return result, err
}

// FindByObjectID implements DbRepository.
func (repo *CachedDbRepository) FindByObjectID(ctx context.Context, objectID string) (*openapi.RouteMetadata, error) {
	result, err, _ := memoize.Call(repo.cache, objectID, func() (*openapi.RouteMetadata, error) {
		return repo.wrapped.FindByObjectID(ctx, objectID)
	})
	return result, err
}

// SearchHits implements DbRepository.
func (repo *CachedDbRepository) SearchHits(ctx context.Context, criteria *openapi.SearchRequest) (*[]openapi.RouteSummary, error) {
	key, err := buildCacheKey(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache key: %v", err)
	}
	result, err, _ := memoize.Call(repo.cache, key, func() (*[]openapi.RouteSummary, error) {
		return repo.wrapped.SearchHits(ctx, criteria)
	})
	return result, err
}

// Store implements DbRepository.
func (repo *CachedDbRepository) Store(ctx context.Context, route *openapi.RouteMetadata) error {
	return repo.wrapped.Store(ctx, route)
}
