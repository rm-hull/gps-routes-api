package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	model "github.com/rm-hull/gps-routes-api/go"
)

type PostgresDbRepository struct {
	pool *pgxpool.Pool
	schema string
}

func NewPostgresRouteRepository(pool *pgxpool.Pool, schema string) *PostgresDbRepository {
	return &PostgresDbRepository{pool: pool, schema: schema}
}

func (repo *PostgresDbRepository) Store(ctx context.Context, route *model.RouteMetadata) error {

	batch := &pgx.Batch{}

	batch.Queue(fmt.Sprintf(`
		INSERT INTO "%s"."routes" (
		    object_id, created_at, ref, title, headline_image_url,
		    gpx_url, _geoloc, distance_km, description, video_url,
		    postcode, district, county, region, country
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, $11,
			$12, $13, $14, $15, $16
		)
		ON CONFLICT (object_id) DO UPDATE SET
			created_at = EXCLUDED.created_at, ref = EXCLUDED.ref,
			title = EXCLUDED.title, headline_image_url = EXCLUDED.headline_image_url,
			gpx_url = EXCLUDED.gpx_url, _geoloc = EXCLUDED._geoloc,
			distance_km = EXCLUDED.distance_km, description = EXCLUDED.description,
			video_url = EXCLUDED.video_url, postcode = EXCLUDED.postcode,
			district = EXCLUDED.district, county = EXCLUDED.county,
			region = EXCLUDED.region, country = EXCLUDED.country`, repo.schema),
		route.ObjectID, route.CreatedAt, route.Ref, route.Title, route.HeadlineImageUrl,
		route.GpxUrl, route.Location.Longitude, route.Location.Latitude, route.DistanceKm, route.Description, route.VideoUrl,
		route.Postcode, route.District, route.County, route.Region, route.Country,
	)

	batch.Queue(
		fmt.Sprintf(`DELETE FROM "%s"."nearby" WHERE route_object_id = $1`, repo.schema),
		route.ObjectID,
	)

	for _, nearby := range route.Nearby {
		batch.Queue(fmt.Sprintf(`
			INSERT INTO "%s"."nearby" (route_object_id, description, object_id, ref)
			VALUES ($1, $2, $3, $4)`, repo.schema),
			route.ObjectID, nearby.Description, nearby.ObjectID, nearby.Ref,
		)
	}

	batch.Queue(
		fmt.Sprintf(`DELETE FROM "%s"."images" WHERE route_object_id = $1`, repo.schema),
		route.ObjectID,
	)

	for _, image := range route.Images {
		batch.Queue(fmt.Sprintf(`
			INSERT INTO "%s"."images" (route_object_id, src, title, caption)
			VALUES ($1, $2, $3, $4)`, repo.schema),
			route.ObjectID, image.Src, image.Title, image.Caption,
		)
	}

	results := repo.pool.SendBatch(ctx, batch)
	defer results.Close()

	// Ensure all queries in the batch suceed
	for i := range batch.Len() {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch insert failed at query %d: %v", i, err)
		}
	}

	return nil
}

func (repo *PostgresDbRepository) FindByObjectID(ctx context.Context, objectID string) (*model.RouteMetadata, error) {

	mainQuery := `
		SELECT
			r.object_id, r.ref, r.title, r.description, r.headline_image_url,
			ST_X(r._geoloc) AS longitude, ST_Y(r._geoloc) AS latitude,
			r.created_at, r.gpx_url, r.distance_km, r.video_url,
			r.postcode, r.district, r.county, r.region, r.country
		FROM routes r
		WHERE r.object_id = $1`

	var route model.RouteMetadata
	var latitude, longitude float64

	// Execute the query and scan the result into variables.
	err := repo.pool.QueryRow(ctx, mainQuery, objectID).Scan(
		&route.ObjectID, &route.Ref, &route.Title, &route.Description,
		&route.HeadlineImageUrl, &longitude, &latitude, &route.CreatedAt,
		&route.GpxUrl, &route.DistanceKm, &route.VideoUrl, &route.Postcode,
		&route.District, &route.County, &route.Region, &route.Country,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route with ObjectID %s: %v", objectID, err)
	}

	// Map the location to the GeoLoc struct.
	route.Location = model.GeoLoc{
		Latitude:  latitude,
		Longitude: longitude,
	}

	// Query images
	imageQuery := `
		SELECT src, title, caption
		FROM images
		WHERE route_object_id = $1`

	rows, err := repo.pool.Query(ctx, imageQuery, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch images: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var image model.Image
		if err := rows.Scan(&image.Src, &image.Title, &image.Caption); err != nil {
			return nil, fmt.Errorf("failed to scan image: %v", err)
		}
		route.Images = append(route.Images, image)
	}

	// Query nearby
	nearbyQuery := `
		SELECT description, object_id, ref
		FROM nearby
		WHERE route_object_id = $1`

	rows, err = repo.pool.Query(ctx, nearbyQuery, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nearby: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var nearby model.Nearby
		if err := rows.Scan(&nearby.Description, &nearby.ObjectID, &nearby.Ref); err != nil {
			return nil, fmt.Errorf("failed to scan nearby: %w", err)
		}
		route.Nearby = append(route.Nearby, nearby)
	}
	return &route, nil
}

// CountAll implements DbRepository.
func (repo *PostgresDbRepository) CountAll(ctx context.Context, criteria *model.SearchRequest) (int64, error) {
	panic("unimplemented")
}

// FacetCounts implements DbRepository.
func (repo *PostgresDbRepository) FacetCounts(ctx context.Context, criteria *model.SearchRequest, facetField string, limit int) (*map[string]int64, error) {
	panic("unimplemented")
}

// SearchHits implements DbRepository.
func (repo *PostgresDbRepository) SearchHits(ctx context.Context, criteria *model.SearchRequest) (*[]model.RouteSummary, error) {
	panic("unimplemented")
}
