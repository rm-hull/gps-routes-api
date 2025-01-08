package repositories

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Depado/ginprom"
	"github.com/kofalt/go-memoize"

	openapi "github.com/rm-hull/gps-routes-api/go"
)

type CachedDbRepository struct {
	cache      *memoize.Memoizer
	prometheus *ginprom.Prometheus
	wrapped    DbRepository
}

// updateMetrics updates the Prometheus metrics for cache hits and misses.
func (repo *CachedDbRepository) updateMetrics(method string, cached bool) {
	repo.prometheus.SetGaugeValue("repo_cache_size", []string{}, float64(repo.cache.Storage.ItemCount()))
	if cached {
		repo.prometheus.IncrementCounterValue("repo_cache_stats_total", []string{method, "hit"})
	} else {
		repo.prometheus.IncrementCounterValue("repo_cache_stats_total", []string{method, "miss"})
	}
}

func NewCachedRepository(prometheus *ginprom.Prometheus, wrapped DbRepository) *CachedDbRepository {
	prometheus.AddCustomCounter("repo_cache_stats_total", "Repo cache statistics (hits & misses)", []string{"method", "type"})
	prometheus.AddCustomGauge("repo_cache_size", "Repo cache size (number of items)", []string{})

	return &CachedDbRepository{
		prometheus: prometheus,
		cache:      memoize.NewMemoizer(90*time.Second, 10*time.Minute),
		wrapped:    wrapped,
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
	result, err, cached := memoize.Call(repo.cache, key, func() (int64, error) {
		return repo.wrapped.CountAll(ctx, criteria)
	})
	repo.updateMetrics("CountAll", cached)
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
	result, err, cached := memoize.Call(repo.cache, key, func() (*map[string]int64, error) {
		return repo.wrapped.FacetCounts(ctx, criteria, facetField, limit)
	})
	repo.updateMetrics("FacetCounts", cached)
	return result, err
}

// FindByObjectID implements DbRepository.
func (repo *CachedDbRepository) FindByObjectID(ctx context.Context, objectID string) (*openapi.RouteMetadata, error) {
	result, err, cached := memoize.Call(repo.cache, objectID, func() (*openapi.RouteMetadata, error) {
		return repo.wrapped.FindByObjectID(ctx, objectID)
	})
	repo.updateMetrics("FindByObjectID", cached)
	return result, err
}

// SearchHits implements DbRepository.
func (repo *CachedDbRepository) SearchHits(ctx context.Context, criteria *openapi.SearchRequest) (*[]openapi.RouteSummary, error) {
	key, err := buildCacheKey(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache key: %v", err)
	}
	result, err, cached := memoize.Call(repo.cache, key, func() (*[]openapi.RouteSummary, error) {
		return repo.wrapped.SearchHits(ctx, criteria)
	})
	repo.updateMetrics("SearchHits", cached)
	return result, err
}

// Store implements DbRepository.
func (repo *CachedDbRepository) Store(ctx context.Context, route *openapi.RouteMetadata) error {
	return repo.wrapped.Store(ctx, route)
}
