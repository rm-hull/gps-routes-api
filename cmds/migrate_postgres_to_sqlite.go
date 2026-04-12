package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	pgConnStr := flag.String("pg-url", "", "PostgreSQL connection string (e.g., user=postgres password=pwd host=localhost dbname=routes)")
	sqliteFile := flag.String("sqlite-db", "./routes_migrated.db", "SQLite database file path")
	dryRun := flag.Bool("dry-run", false, "Perform validation without writing to SQLite")
	maxRecords := flag.Int("max-records", 0, "Maximum records to migrate (0 = all)")
	flag.Parse()

	if *pgConnStr == "" {
		fmt.Println("Usage: migrate_postgres_to_sqlite -pg-url <connection_string> -sqlite-db <path>")
		fmt.Println("\nExample:")
		fmt.Println("  migrate_postgres_to_sqlite \\")
		fmt.Println("    -pg-url 'user=postgres password=pwd host=localhost dbname=routes' \\")
		fmt.Println("    -sqlite-db ./routes.db")
		os.Exit(1)
	}

	log.Printf("🔄 SQLite3 Migration Tool")
	log.Printf("📊 Source (PostgreSQL): %s", maskConnStr(*pgConnStr))
	log.Printf("📁 Target (SQLite): %s", *sqliteFile)
	if *dryRun {
		log.Printf("⚠️  DRY RUN: Validation only, no writes")
	}

	// Connect to PostgreSQL
	pgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgPool, err := pgxpool.New(pgCtx, *pgConnStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	log.Print("✅ Connected to PostgreSQL")

	// Validate PostgreSQL has data
	var pgCount int64
	err = pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM routes").Scan(&pgCount)
	if err != nil {
		log.Fatalf("❌ Failed to query PostgreSQL: %v", err)
	}
	log.Printf("📈 PostgreSQL routes count: %d", pgCount)

	if pgCount == 0 {
		log.Fatal("❌ No routes found in PostgreSQL database")
	}

	if *maxRecords > 0 && int64(*maxRecords) < pgCount {
		log.Printf("⏱️  Limiting migration to %d records (out of %d available)", *maxRecords, pgCount)
		pgCount = int64(*maxRecords)
	}

	// Create/prepare SQLite database
	if err := setupSQLiteDB(*sqliteFile, *dryRun); err != nil {
		log.Fatalf("❌ Failed to setup SQLite: %v", err)
	}

	if *dryRun {
		log.Print("🔍 Dry-run mode: skipping data migration")
		os.Exit(0)
	}

	// Migrate routes
	sqliteDB, err := sql.Open("sqlite3", *sqliteFile)
	if err != nil {
		log.Fatalf("❌ Failed to open SQLite: %v", err)
	}
	defer sqliteDB.Close()

	log.Print("🚀 Starting data migration...")
	migrated, err := migrateRoutes(pgPool, sqliteDB, *maxRecords)
	if err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	log.Printf("✅ Routes migrated: %d", migrated)

	// Migrate related tables
	nearbyMigrated, err := migrateNearby(pgPool, sqliteDB)
	if err != nil {
		log.Printf("⚠️  Nearby migration failed (non-critical): %v", err)
	} else {
		log.Printf("✅ Nearby records migrated: %d", nearbyMigrated)
	}

	imagesMigrated, err := migrateImages(pgPool, sqliteDB)
	if err != nil {
		log.Printf("⚠️  Images migration failed (non-critical): %v", err)
	} else {
		log.Printf("✅ Images migrated: %d", imagesMigrated)
	}

	detailsMigrated, err := migrateDetails(pgPool, sqliteDB)
	if err != nil {
		log.Printf("⚠️  Details migration failed (non-critical): %v", err)
	} else {
		log.Printf("✅ Details migrated: %d", detailsMigrated)
	}

	// Validate counts match
	log.Print("🔍 Validating migration...")
	if err := validateMigration(pgPool, sqliteDB); err != nil {
		log.Fatalf("❌ Validation failed: %v", err)
	}

	log.Print("✅ Migration validation passed!")
	log.Printf("📊 Final SQLite database: %s", *sqliteFile)
}

func setupSQLiteDB(dbPath string, dryRun bool) error {
	if dryRun {
		// Use in-memory DB for dry-run
		dbPath = ":memory:"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Read and execute schema
	schema, err := os.ReadFile("db/migrations/00002_sqlite_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema creation
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func migrateRoutes(pgPool *pgxpool.Pool, sqliteDB *sql.DB, maxRecords int) (int64, error) {
	ctx := context.Background()

	// Fetch from PostgreSQL (with limit if specified)
	query := `
		SELECT
			object_id, created_at, ref, title, headline_image_url, gpx_url,
			ST_X(_geoloc) as longitude, ST_Y(_geoloc) as latitude,
			distance_km, description, video_url, display_address, postcode,
			district, county, region, state, country, estimated_duration,
			difficulty, terrain, points_of_interest, facilities, route_type, activities
		FROM routes
	`
	if maxRecords > 0 {
		query += fmt.Sprintf(" LIMIT %d", maxRecords)
	}

	rows, err := pgPool.Query(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query PostgreSQL routes: %w", err)
	}
	defer rows.Close()

	// Prepare SQLite insert statement
	stmt, err := sqliteDB.Prepare(`
		INSERT INTO routes (
			object_id, created_at, ref, title, headline_image_url, gpx_url,
			latitude, longitude, distance_km, description, video_url,
			display_address, postcode, district, county, region, state, country,
			estimated_duration, difficulty, terrain, points_of_interest,
			facilities, route_type, activities
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare SQLite statement: %w", err)
	}
	defer stmt.Close()

	// Begin transaction
	tx, err := sqliteDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var count int64
	for rows.Next() {
		var objectID, ref, title, description, createdAt, estimatedDuration, difficulty string
		var headlineImageURL, gpxURL, videoURL, displayAddress, postcode string
		var district, county, region, state, country, routeType sql.NullString
		var longitude, latitude, distanceKm sql.NullFloat64
		var terrain, poi, facilities, activities sql.NullString

		err := rows.Scan(
			&objectID, &createdAt, &ref, &title, &headlineImageURL, &gpxURL,
			&longitude, &latitude, &distanceKm, &description, &videoURL,
			&displayAddress, &postcode, &district, &county, &region, &state,
			&country, &estimatedDuration, &difficulty, &terrain, &poi,
			&facilities, &routeType, &activities,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert PostgreSQL arrays to JSON arrays
		terrainJSON := arrayToJSON(terrain.String)
		poiJSON := arrayToJSON(poi.String)
		facilitiesJSON := arrayToJSON(facilities.String)
		activitiesJSON := arrayToJSON(activities.String)

		_, err = stmt.Exec(
			objectID, createdAt, ref, title, headlineImageURL, gpxURL,
			latitude, longitude, distanceKm, description, videoURL,
			displayAddress, postcode, district, county, region, state,
			country, estimatedDuration, difficulty, terrainJSON, poiJSON,
			facilitiesJSON, routeType, activitiesJSON,
		)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to insert route %s: %w", objectID, err)
		}

		count++
		if count%1000 == 0 {
			log.Printf("  ⏳ Migrated %d routes...", count)
		}
	}

	if rows.Err() != nil {
		tx.Rollback()
		return 0, fmt.Errorf("query error: %w", rows.Err())
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func migrateNearby(pgPool *pgxpool.Pool, sqliteDB *sql.DB) (int64, error) {
	ctx := context.Background()
	rows, err := pgPool.Query(ctx, "SELECT route_object_id, description, object_id, ref FROM nearby")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	stmt, err := sqliteDB.Prepare("INSERT INTO nearby (route_object_id, description, object_id, ref) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int64
	for rows.Next() {
		var routeID, description, objectID, ref string
		if err := rows.Scan(&routeID, &description, &objectID, &ref); err != nil {
			return 0, err
		}
		if _, err := stmt.Exec(routeID, description, objectID, ref); err != nil {
			return 0, err
		}
		count++
	}
	return count, rows.Err()
}

func migrateImages(pgPool *pgxpool.Pool, sqliteDB *sql.DB) (int64, error) {
	ctx := context.Background()
	rows, err := pgPool.Query(ctx, "SELECT route_object_id, src, title, caption FROM images")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	stmt, err := sqliteDB.Prepare("INSERT INTO images (route_object_id, src, title, caption) VALUES (?, ?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int64
	for rows.Next() {
		var routeID, src, title, caption string
		if err := rows.Scan(&routeID, &src, &title, &caption); err != nil {
			return 0, err
		}
		if _, err := stmt.Exec(routeID, src, title, caption); err != nil {
			return 0, err
		}
		count++
	}
	return count, rows.Err()
}

func migrateDetails(pgPool *pgxpool.Pool, sqliteDB *sql.DB) (int64, error) {
	ctx := context.Background()
	rows, err := pgPool.Query(ctx, "SELECT route_object_id, subtitle, content FROM details")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	stmt, err := sqliteDB.Prepare("INSERT INTO details (route_object_id, subtitle, content) VALUES (?, ?, ?)")
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int64
	for rows.Next() {
		var routeID, subtitle, content string
		if err := rows.Scan(&routeID, &subtitle, &content); err != nil {
			return 0, err
		}
		if _, err := stmt.Exec(routeID, subtitle, content); err != nil {
			return 0, err
		}
		count++
	}
	return count, rows.Err()
}

func validateMigration(pgPool *pgxpool.Pool, sqliteDB *sql.DB) error {
	ctx := context.Background()

	// Compare record counts
	var pgRouteCount int64
	err := pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM routes").Scan(&pgRouteCount)
	if err != nil {
		return fmt.Errorf("failed to get PG count: %w", err)
	}

	var sqliteRouteCount int64
	err = sqliteDB.QueryRow("SELECT COUNT(*) FROM routes").Scan(&sqliteRouteCount)
	if err != nil {
		return fmt.Errorf("failed to get SQLite count: %w", err)
	}

	if pgRouteCount != sqliteRouteCount {
		return fmt.Errorf("route count mismatch: PostgreSQL=%d, SQLite=%d", pgRouteCount, sqliteRouteCount)
	}

	log.Printf("✅ Record counts match: %d routes", sqliteRouteCount)
	return nil
}

// arrayToJSON converts PostgreSQL array string to JSON array
// Input: "{hiking,mountain}" → Output: "["hiking","mountain"]"
func arrayToJSON(pgArray string) string {
	if pgArray == "" || pgArray == "{}" {
		return "[]"
	}

	// Remove braces
	pgArray = pgArray[1 : len(pgArray)-1]

	// Split by comma and quote each element
	elements := make([]string, 0)
	var current string
	for _, char := range pgArray {
		if char == ',' {
			if current != "" {
				elements = append(elements, fmt.Sprintf(`"%s"`, current))
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		elements = append(elements, fmt.Sprintf(`"%s"`, current))
	}

	// Build JSON array
	jsonArray := "["
	for i, elem := range elements {
		jsonArray += elem
		if i < len(elements)-1 {
			jsonArray += ","
		}
	}
	jsonArray += "]"

	return jsonArray
}

func maskConnStr(connStr string) string {
	// Mask password for logging
	if len(connStr) > 20 {
		return connStr[:10] + "***" + connStr[len(connStr)-10:]
	}
	return "***"
}
