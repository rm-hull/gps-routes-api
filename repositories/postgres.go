package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rm-hull/gps-routes-api/db"
	model "github.com/rm-hull/gps-routes-api/go"
)

type PostgresDbRepository struct {
	pool   *pgxpool.Pool
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
		    display_address, postcode, district, county, region,
			state, country
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18
		)
		ON CONFLICT (object_id) DO UPDATE SET
			created_at = EXCLUDED.created_at, ref = EXCLUDED.ref,
			title = EXCLUDED.title, headline_image_url = EXCLUDED.headline_image_url,
			gpx_url = EXCLUDED.gpx_url, _geoloc = EXCLUDED._geoloc,
			distance_km = EXCLUDED.distance_km, description = EXCLUDED.description,
			video_url = EXCLUDED.video_url, display_address = EXCLUDED.display_address,
			postcode = EXCLUDED.postcode, district = EXCLUDED.district, county = EXCLUDED.county,
			region = EXCLUDED.region, state = EXCLUDED.state, country = EXCLUDED.country`, repo.schema),
		route.ObjectID, route.CreatedAt, route.Ref, route.Title, route.HeadlineImageUrl, route.GpxUrl,
		route.StartPosition.Longitude, route.StartPosition.Latitude, route.DistanceKm, route.Description,
		route.VideoUrl, route.DisplayAddress, route.Postcode, route.District, route.County, route.Region,
		route.State, route.Country,
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

	batch.Queue(
		fmt.Sprintf(`DELETE FROM "%s"."details" WHERE route_object_id = $1`, repo.schema),
		route.ObjectID,
	)

	for _, detail := range route.Details {
		batch.Queue(fmt.Sprintf(`
			INSERT INTO "%s"."details" (route_object_id, subtitle, content)
			VALUES ($1, $2, $3)`, repo.schema),
			route.ObjectID, detail.Subtitle, detail.Content,
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
			object_id, ref, title, description, headline_image_url,
			ST_X(_geoloc) AS longitude, ST_Y(_geoloc) AS latitude,
			created_at, gpx_url, distance_km, video_url,
			display_address, postcode, district, county,
			region, state, country
		FROM routes
		WHERE object_id = $1`

	var route model.RouteMetadata
	var latitude, longitude float64

	// Execute the query and scan the result into variables.
	err := repo.pool.QueryRow(ctx, mainQuery, objectID).Scan(
		&route.ObjectID, &route.Ref, &route.Title, &route.Description,
		&route.HeadlineImageUrl, &longitude, &latitude, &route.CreatedAt,
		&route.GpxUrl, &route.DistanceKm, &route.VideoUrl, &route.DisplayAddress,
		&route.Postcode, &route.District, &route.County, &route.Region,
		&route.State, &route.Country,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scan query result: %v", err)
	}

	// Map the start position to the GeoLoc struct.
	route.StartPosition = model.GeoLoc{
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
		return nil, fmt.Errorf("failed to fetch nearby: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var nearby model.Nearby
		if err := rows.Scan(&nearby.Description, &nearby.ObjectID, &nearby.Ref); err != nil {
			return nil, fmt.Errorf("failed to scan nearby: %v", err)
		}
		route.Nearby = append(route.Nearby, nearby)
	}

	// Query details
	detailsQuery := `
		SELECT subtitle, content
		FROM details
		WHERE route_object_id = $1`

	rows, err = repo.pool.Query(ctx, detailsQuery, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch details: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var detail model.Detail
		if err := rows.Scan(&detail.Subtitle, &detail.Content); err != nil {
			return nil, fmt.Errorf("failed to scan detail: %v", err)
		}
		route.Details = append(route.Details, detail)
	}
	return &route, nil
}

func (repo *PostgresDbRepository) CountAll(ctx context.Context, criteria *model.SearchRequest) (int64, error) {

	var count int64
	query, params := db.NewQueryBuilder(`SELECT count(1) FROM routes`, criteria).Build()
	err := repo.pool.QueryRow(ctx, query, params...).Scan(&count)
	return count, err
}

func (repo *PostgresDbRepository) FacetCounts(ctx context.Context, criteria *model.SearchRequest, facetField string, limit int32) (*map[string]int64, error) {
	results := make(map[string]int64, 0)

	selectPart := fmt.Sprintf(`SELECT %s, COUNT(*) AS value FROM routes`, facetField)
	query, params := db.NewQueryBuilder(selectPart, criteria).
		WithWhereClause(fmt.Sprintf("%s IS NOT NULL", facetField)).
		WithGroupBy(facetField).
		WithOrderBy("value").
		WithLimit(limit).
		Build()

	rows, err := repo.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s-facet query: %v", facetField, err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int

		err := rows.Scan(&name, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s-facet row: %v", facetField, err)
		}

		results[name] = int64(value)
	}

	return &results, nil
}

func (repo *PostgresDbRepository) SearchHits(ctx context.Context, criteria *model.SearchRequest) (*[]model.RouteSummary, error) {

	selectPart := `
		SELECT
			object_id, ref,
			CASE
				WHEN LENGTH(title) > 50 THEN SUBSTRING(title, 1, 47) || '...'
				ELSE title
			END AS title,
			CASE
				WHEN LENGTH(description) > 150 THEN SUBSTRING(description, 1, 147) || '...'
				ELSE description
			END AS description,
			headline_image_url,
			ST_X(_geoloc) AS longitude,
			ST_Y(_geoloc) AS latitude,
			distance_km
		FROM routes`

	sortField := "created_at"
	if criteria.Query != "" {
		sortField = "ts_rank_cd(search_vector, to_tsquery($1), 32)"
	}

	query, params := db.NewQueryBuilder(selectPart, criteria).
		WithOrderBy(sortField).
		WithOffset(criteria.Offset).
		WithLimit(criteria.Limit).
		Build()

	results := make([]model.RouteSummary, 0)
	rows, err := repo.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var summary model.RouteSummary
		var latitude, longitude float64

		err := rows.Scan(
			&summary.ObjectID, &summary.Ref, &summary.Title,
			&summary.Description, &summary.HeadlineImageUrl,
			&longitude, &latitude, &summary.DistanceKm)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %v", err)
		}

		summary.StartPosition = model.GeoLoc{
			Latitude:  latitude,
			Longitude: longitude,
		}

		results = append(results, summary)
	}
	return &results, nil
}
