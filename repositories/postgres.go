package repositories

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rm-hull/gps-routes-api/db"
	"github.com/rm-hull/gps-routes-api/models/common"
	"github.com/rm-hull/gps-routes-api/models/domain"
	"github.com/rm-hull/gps-routes-api/models/request"
)

type PostgresDbRepository struct {
	pool   *pgxpool.Pool
	schema string
}

var DEFAULT_ARRAY_FIELDS = []string{"activities", "terrain", "facilities", "points_of_interest"}

func NewPostgresRouteRepository(pool *pgxpool.Pool, schema string) *PostgresDbRepository {
	return &PostgresDbRepository{pool: pool, schema: schema}
}

func (repo *PostgresDbRepository) Store(ctx context.Context, route *domain.RouteMetadata) error {

	batch := &pgx.Batch{}

	batch.Queue(fmt.Sprintf(`
		INSERT INTO "%s"."routes" (
		    object_id, created_at, ref, title, headline_image_url,
		    gpx_url, _geoloc, distance_km, description, video_url,
		    display_address, postcode, district, county, region,
			state, country, estimated_duration, difficulty, terrain,
			points_of_interest, facilities, route_type, activities
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21,
			$22, $23, $24, $25
		)
		ON CONFLICT (object_id) DO UPDATE SET
			created_at = EXCLUDED.created_at, ref = EXCLUDED.ref,
			title = EXCLUDED.title, headline_image_url = EXCLUDED.headline_image_url,
			gpx_url = EXCLUDED.gpx_url, _geoloc = EXCLUDED._geoloc,
			distance_km = EXCLUDED.distance_km, description = EXCLUDED.description,
			video_url = EXCLUDED.video_url, display_address = EXCLUDED.display_address,
			postcode = EXCLUDED.postcode, district = EXCLUDED.district, county = EXCLUDED.county,
			region = EXCLUDED.region, state = EXCLUDED.state, country = EXCLUDED.country,
			estimated_duration = EXCLUDED.estimated_duration, difficulty = EXCLUDED.difficulty,
			terrain = EXCLUDED.terrain, points_of_interest = EXCLUDED.points_of_interest,
			facilities = EXCLUDED.facilities, route_type = EXCLUDED.route_type,
			activities = EXCLUDED.activities`, repo.schema),
		route.ObjectID, route.CreatedAt, route.Ref, route.Title, route.HeadlineImageUrl, route.GpxUrl,
		route.StartPosition.Longitude, route.StartPosition.Latitude, route.DistanceKm, route.Description,
		route.VideoUrl, route.DisplayAddress, route.Postcode, route.District, route.County, route.Region,
		route.State, route.Country, route.EstimatedDuration, route.Difficulty, route.Terrain,
		route.PointsOfInterest, route.Facilities, route.RouteType, route.Activities,
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
	defer func() {
		if err := results.Close(); err != nil {
			log.Printf("Error closing batch results: %v", err)
		}
	}()

	// Ensure all queries in the batch succeed
	for i := range batch.Len() {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch insert failed at query %d: %v", i, err)
		}
	}

	return nil
}

func (repo *PostgresDbRepository) FindByObjectID(ctx context.Context, objectID string) (*domain.RouteMetadata, error) {

	mainQuery := `
		SELECT
			object_id, ref, title, description, headline_image_url,
			ST_X(_geoloc) AS longitude, ST_Y(_geoloc) AS latitude,
			created_at, gpx_url, distance_km, video_url,
			display_address, postcode, district, county,
			region, state, country, estimated_duration,
			difficulty, terrain, points_of_interest,
			facilities, route_type, activities
		FROM routes
		WHERE object_id = $1`

	var route domain.RouteMetadata
	var latitude, longitude float64

	// Execute the query and scan the result into variables.
	err := repo.pool.QueryRow(ctx, mainQuery, objectID).Scan(
		&route.ObjectID, &route.Ref, &route.Title, &route.Description,
		&route.HeadlineImageUrl, &longitude, &latitude, &route.CreatedAt,
		&route.GpxUrl, &route.DistanceKm, &route.VideoUrl, &route.DisplayAddress,
		&route.Postcode, &route.District, &route.County, &route.Region,
		&route.State, &route.Country, &route.EstimatedDuration, &route.Difficulty,
		&route.Terrain, &route.PointsOfInterest, &route.Facilities, &route.RouteType,
		&route.Activities,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to scan query result: %v", err)
	}

	// Map the start position to the GeoLoc struct.
	route.StartPosition = common.GeoLoc{
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
		var image domain.Image
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
		var nearby domain.Nearby
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
		var detail domain.Detail
		if err := rows.Scan(&detail.Subtitle, &detail.Content); err != nil {
			return nil, fmt.Errorf("failed to scan detail: %v", err)
		}
		route.Details = append(route.Details, detail)
	}
	return &route, nil
}

func (repo *PostgresDbRepository) CountAll(ctx context.Context, criteria *request.SearchRequest) (int64, error) {

	var count int64
	query, params := db.NewQueryBuilder(`SELECT count(1) FROM routes`, criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		Build()

	err := repo.pool.QueryRow(ctx, query, params...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count-all query \"%s\": %v", query, err)
	}

	return count, nil
}

func (repo *PostgresDbRepository) FacetCounts(ctx context.Context, criteria *request.SearchRequest, facetField string, limit int32, unnest bool, excludeFacets ...string) (*map[string]int64, error) {
	results := make(map[string]int64, 0)

	format := map[bool]string{
		true:  `SELECT UNNEST(%s) AS key, COUNT(*) AS value FROM routes`,
		false: `SELECT %s AS key, COUNT(*) AS value FROM routes`,
	}[unnest]

	query, params := db.NewQueryBuilder(fmt.Sprintf(format, facetField), criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		WithWhereClause(fmt.Sprintf("%s IS NOT NULL", facetField)).
		WithExcludeFacets(excludeFacets...).
		WithGroupBy("key").
		WithOrderBy("value DESC").
		WithLimit(limit).
		Build()

	rows, err := repo.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s-facet query \"%s\": %v", facetField, query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int

		err = rows.Scan(&name, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s-facet row: %v", facetField, err)
		}

		results[name] = int64(value)
	}

	return &results, nil
}

func (repo *PostgresDbRepository) SearchHits(ctx context.Context, criteria *request.SearchRequest) (*[]domain.RouteSummary, error) {

	titleDescription := "title, description"
	if criteria.TruncateText {
		titleDescription = `
			CASE
				WHEN LENGTH(title) > 50 THEN SUBSTRING(title, 1, 49) || '…'
				ELSE title
			END AS title,
			CASE
				WHEN LENGTH(description) > 150 THEN SUBSTRING(description, 1, 149) || '…'
				ELSE description
			END AS description`
	}

	selectPart := fmt.Sprintf(`
		SELECT
			object_id, ref,
			%s,
			headline_image_url,
			ST_X(_geoloc) AS longitude,
			ST_Y(_geoloc) AS latitude,
			distance_km
		FROM routes`, titleDescription)

	sortField := "created_at DESC"
	// FIXME: investigate why this sort field doesnt seem to work
	// ----------------------------------------------------------
	// if criteria.Nearby != nil && criteria.Nearby.Center != nil {
	// 	sortField = fmt.Sprintf("_geoloc <-> ST_Point(%f, %f)", criteria.Nearby.Center.Longitude, criteria.Nearby.Center.Latitude)
	// } else
	if criteria.Query != "" {
		sortField = "ts_rank_cd(search_vector, to_tsquery($1), 32) DESC"
	}

	query, params := db.NewQueryBuilder(selectPart, criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		WithOrderBy(sortField).
		WithOffset(criteria.Offset).
		WithLimit(criteria.Limit).
		Build()

	rows, err := repo.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query \"%s\":, %v", query, err)
	}
	defer rows.Close()

	results := make([]domain.RouteSummary, 0, criteria.Limit)
	var summary domain.RouteSummary
	var latitude, longitude float64

	for rows.Next() {
		err := rows.Scan(
			&summary.ObjectID, &summary.Ref, &summary.Title,
			&summary.Description, &summary.HeadlineImageUrl,
			&longitude, &latitude, &summary.DistanceKm)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %v", err)
		}

		summary.StartPosition = common.GeoLoc{
			Latitude:  latitude,
			Longitude: longitude,
		}

		results = append(results, summary)
	}
	return &results, nil
}
