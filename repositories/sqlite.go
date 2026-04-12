//go:build !postgres

package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rm-hull/gps-routes-api/db"
	"github.com/rm-hull/gps-routes-api/models/common"
	"github.com/rm-hull/gps-routes-api/models/domain"
	"github.com/rm-hull/gps-routes-api/models/request"
)

type SqliteDbRepository struct {
	db     *sql.DB
	schema string
}

var DEFAULT_ARRAY_FIELDS = []string{"activities", "terrain", "facilities", "points_of_interest"}

func NewRepository(pool *pgxpool.Pool, db *sql.DB, config db.DBConfig) DbRepository {
	return NewSqliteRouteRepository(db, config.Schema)
}

func NewSqliteRouteRepository(db *sql.DB, schema string) *SqliteDbRepository {
	return &SqliteDbRepository{db: db, schema: schema}
}

func (repo *SqliteDbRepository) Store(ctx context.Context, route *domain.RouteMetadata) error {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelImmediate})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	dialect := &db.SQLiteDialect{}

	// Upsert route
	// Note: Spatialite uses ST_Point(lon, lat)
	query := `
		INSERT INTO "routes" (
		    object_id, created_at, ref, title, headline_image_url,
		    gpx_url, _geoloc, distance_km, description, video_url,
		    display_address, postcode, district, county, region,
			state, country, estimated_duration, difficulty, terrain,
			points_of_interest, facilities, route_type, activities
		) VALUES (
			?, ?, ?, ?, ?,
			?, ST_SetSRID(ST_Point(?, ?), 4326), ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?
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
			activities = EXCLUDED.activities`

	_, err = tx.ExecContext(ctx, query,
		route.ObjectID, route.CreatedAt, route.Ref, route.Title, route.HeadlineImageUrl, route.GpxUrl,
		route.StartPosition.Longitude, route.StartPosition.Latitude, route.DistanceKm, route.Description,
		route.VideoUrl, route.DisplayAddress, route.Postcode, route.District, route.County, route.Region,
		route.State, route.Country, route.EstimatedDuration, route.Difficulty,
		dialect.PrepareParam(route.Terrain),
		dialect.PrepareParam(route.PointsOfInterest),
		dialect.PrepareParam(route.Facilities),
		route.RouteType,
		dialect.PrepareParam(route.Activities),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert route: %w", err)
	}

	// Delete and re-insert nearby
	_, err = tx.ExecContext(ctx, `DELETE FROM "nearby" WHERE route_object_id = ?`, route.ObjectID)
	if err != nil {
		return fmt.Errorf("failed to delete nearby: %w", err)
	}

	for _, nearby := range route.Nearby {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO "nearby" (route_object_id, description, object_id, ref)
			VALUES (?, ?, ?, ?)`,
			route.ObjectID, nearby.Description, nearby.ObjectID, nearby.Ref,
		)
		if err != nil {
			return fmt.Errorf("failed to insert nearby: %w", err)
		}
	}

	// Delete and re-insert images
	_, err = tx.ExecContext(ctx, `DELETE FROM "images" WHERE route_object_id = ?`, route.ObjectID)
	if err != nil {
		return fmt.Errorf("failed to delete images: %w", err)
	}

	for _, image := range route.Images {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO "images" (route_object_id, src, title, caption)
			VALUES (?, ?, ?, ?)`,
			route.ObjectID, image.Src, image.Title, image.Caption,
		)
		if err != nil {
			return fmt.Errorf("failed to insert image: %w", err)
		}
	}

	// Delete and re-insert details
	_, err = tx.ExecContext(ctx, `DELETE FROM "details" WHERE route_object_id = ?`, route.ObjectID)
	if err != nil {
		return fmt.Errorf("failed to delete details: %w", err)
	}

	for _, detail := range route.Details {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO "details" (route_object_id, subtitle, content)
			VALUES (?, ?, ?)`,
			route.ObjectID, detail.Subtitle, detail.Content,
		)
		if err != nil {
			return fmt.Errorf("failed to insert detail: %w", err)
		}
	}

	return tx.Commit()
}

func (repo *SqliteDbRepository) FindByObjectID(ctx context.Context, objectID string) (*domain.RouteMetadata, error) {
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
		WHERE object_id = ?`

	var route domain.RouteMetadata
	var latitude, longitude float64
	var terrainJSON, poiJSON, facilitiesJSON, activitiesJSON string

	err := repo.db.QueryRowContext(ctx, mainQuery, objectID).Scan(
		&route.ObjectID, &route.Ref, &route.Title, &route.Description,
		&route.HeadlineImageUrl, &longitude, &latitude, &route.CreatedAt,
		&route.GpxUrl, &route.DistanceKm, &route.VideoUrl, &route.DisplayAddress,
		&route.Postcode, &route.District, &route.County, &route.Region,
		&route.State, &route.Country, &route.EstimatedDuration, &route.Difficulty,
		&terrainJSON, &poiJSON, &facilitiesJSON, &route.RouteType,
		&activitiesJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan route: %w", err)
	}

	route.StartPosition = common.GeoLoc{
		Latitude:  latitude,
		Longitude: longitude,
	}

	// Unmarshal JSON fields
	_ = json.Unmarshal([]byte(terrainJSON), &route.Terrain)
	_ = json.Unmarshal([]byte(poiJSON), &route.PointsOfInterest)
	_ = json.Unmarshal([]byte(facilitiesJSON), &route.Facilities)
	_ = json.Unmarshal([]byte(activitiesJSON), &route.Activities)

	// Query images
	rows, err := repo.db.QueryContext(ctx, `SELECT src, title, caption FROM images WHERE route_object_id = ?`, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch images: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var image domain.Image
		if err := rows.Scan(&image.Src, &image.Title, &image.Caption); err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		route.Images = append(route.Images, image)
	}

	// Query nearby
	nearbySelectPart := `
		SELECT
			r.object_id,
			r.ref,
			r.title,
			r.description,
			r.headline_image_url,
			ST_X(r._geoloc) AS longitude,
			ST_Y(r._geoloc) AS latitude,
			r.distance_km
		FROM nearby n
		INNER JOIN routes r ON n.object_id = r.object_id`

	qb := db.NewQueryBuilder(&db.SQLiteDialect{}, nearbySelectPart, &request.SearchRequest{TruncateText: true}).
		WithTruncatedField("r.title", 50).
		WithTruncatedField("r.description", 150).
		WithWhereClause("n.route_object_id = ?").
		WithParam(objectID)

	nearbyQuery, params := qb.Build()
	rows, err = repo.db.QueryContext(ctx, nearbyQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nearby: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var nearby domain.RouteSummary
		var lat, lng float64
		if err := rows.Scan(&nearby.ObjectID, &nearby.Ref, &nearby.Title, &nearby.Description, &nearby.HeadlineImageUrl, &lng, &lat, &nearby.DistanceKm); err != nil {
			return nil, fmt.Errorf("failed to scan nearby: %w", err)
		}
		nearby.StartPosition = common.GeoLoc{Latitude: lat, Longitude: lng}
		route.Nearby = append(route.Nearby, nearby)
	}

	// Query details
	rows, err = repo.db.QueryContext(ctx, `SELECT subtitle, content FROM details WHERE route_object_id = ?`, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch details: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var detail domain.Detail
		if err := rows.Scan(&detail.Subtitle, &detail.Content); err != nil {
			return nil, fmt.Errorf("failed to scan detail: %w", err)
		}
		route.Details = append(route.Details, detail)
	}

	return &route, nil
}

func (repo *SqliteDbRepository) CountAll(ctx context.Context, criteria *request.SearchRequest) (int64, error) {
	var count int64
	query, params := db.NewQueryBuilder(&db.SQLiteDialect{}, `SELECT count(1) FROM routes`, criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		Build()

	err := repo.db.QueryRowContext(ctx, query, params...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count-all query: %w", err)
	}
	return count, nil
}

func (repo *SqliteDbRepository) FacetCounts(ctx context.Context, criteria *request.SearchRequest, facetField string, limit int32, unnest bool, excludeFacets ...string) (*map[string]int64, error) {
	results := make(map[string]int64)

	dialect := &db.SQLiteDialect{}
	selectPart := dialect.BuildFacetQuery(facetField, unnest)

	query, params := db.NewQueryBuilder(dialect, selectPart, criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		WithWhereClause(fmt.Sprintf("%s IS NOT NULL", facetField)).
		WithExcludeFacets(excludeFacets...).
		WithGroupBy("key").
		WithOrderBy("value DESC").
		WithLimit(limit).
		Build()

	rows, err := repo.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute %s-facet query: %w", facetField, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("failed to scan %s-facet row: %w", facetField, err)
		}
		results[name] = value
	}
	return &results, nil
}

func (repo *SqliteDbRepository) SearchHits(ctx context.Context, criteria *request.SearchRequest) (*[]domain.RouteSummary, error) {
	selectPart := `
		SELECT
			routes.object_id,
			routes.ref,
			routes.title,
			routes.description,
			routes.headline_image_url,
			ST_X(routes._geoloc) AS longitude,
			ST_Y(routes._geoloc) AS latitude,
			routes.distance_km
		FROM routes`

	dialect := &db.SQLiteDialect{}

	if criteria.Query != "" {
		selectPart += " JOIN routes_fts ON routes.rowid = routes_fts.rowid"
	}

	sortField := "created_at DESC"
	if criteria.Nearby != nil && criteria.Nearby.Center != nil {
		sortField = dialect.BuildDistanceSort("routes._geoloc", criteria.Nearby.Center.Longitude, criteria.Nearby.Center.Latitude) + " ASC"
	} else if criteria.Query != "" {
		sortField = dialect.BuildFTSSort("search_vector", "?") + " DESC"
	}

	qb := db.NewQueryBuilder(dialect, selectPart, criteria).
		WithArrayFields(DEFAULT_ARRAY_FIELDS...).
		WithOrderBy(sortField).
		WithOffset(criteria.Offset).
		WithLimit(criteria.Limit)

	if criteria.TruncateText {
		qb.WithTruncatedField("routes.title", 50)
		qb.WithTruncatedField("routes.description", 150)
	}

	query, params := qb.Build()
	rows, err := repo.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	results := make([]domain.RouteSummary, 0, criteria.Limit)
	for rows.Next() {
		var summary domain.RouteSummary
		var lat, lng float64
		err := rows.Scan(
			&summary.ObjectID, &summary.Ref, &summary.Title,
			&summary.Description, &summary.HeadlineImageUrl,
			&lng, &lat, &summary.DistanceKm)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}

		summary.StartPosition = common.GeoLoc{
			Latitude:  lat,
			Longitude: lng,
		}
		results = append(results, summary)
	}
	return &results, nil
}
