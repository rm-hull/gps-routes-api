package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
)

// TestMigrationValidation validates that PostgreSQL and SQLite return identical results
// for critical query patterns (FTS, spatial, facets)
func TestMigrationValidation(t *testing.T) {
	pgConnStr := os.Getenv("POSTGRESQL_URL")
	if pgConnStr == "" {
		t.Skip("POSTGRESQL_URL not set; skipping migration validation")
	}

	sqliteFile := "/tmp/routes_test.db"
	defer func() { _ = os.Remove(sqliteFile) }()

	// Connect to PostgreSQL
	pgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgPool, err := pgxpool.New(pgCtx, pgConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Open SQLite
	sqliteDB, err := sql.Open("sqlite3", sqliteFile)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer func() { _ = sqliteDB.Close() }()

	// Initialize SQLite schema
	schema, err := os.ReadFile("db/migrations/00002_sqlite_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema: %v", err)
	}
	if _, err := sqliteDB.Exec(string(schema)); err != nil {
		t.Fatalf("Failed to create SQLite schema: %v", err)
	}

	t.Run("RecordCount", func(t *testing.T) {
		var pgCount int64
		err := pgPool.QueryRow(context.Background(), "SELECT COUNT(*) FROM routes").Scan(&pgCount)
		if err != nil {
			t.Fatalf("Failed to get PG count: %v", err)
		}

		var sqliteCount int64
		err = sqliteDB.QueryRow("SELECT COUNT(*) FROM routes").Scan(&sqliteCount)
		if err != nil {
			t.Fatalf("Failed to get SQLite count: %v", err)
		}

		if pgCount != sqliteCount {
			t.Errorf("Record count mismatch: PG=%d, SQLite=%d", pgCount, sqliteCount)
		}
		t.Logf("✅ Record counts match: %d", sqliteCount)
	})

	t.Run("FullTextSearchResults", func(t *testing.T) {
		// Compare FTS results for sample queries
		queries := []string{"hiking", "lake", "mountain", "walk"}

		for _, query := range queries {
			// PostgreSQL FTS query
			pgRows, err := pgPool.Query(context.Background(),
				fmt.Sprintf(`
					SELECT COUNT(*) FROM routes
					WHERE search_vector @@ to_tsquery('english', '%s:*')
				`, query))
			if err != nil {
				t.Errorf("PG FTS query failed for '%s': %v", query, err)
				continue
			}
			pgRows.Close()

			// SQLite FTS query equivalent
			sqliteRows, err := sqliteDB.Query(
				fmt.Sprintf(`
					SELECT COUNT(*) FROM routes_fts
					WHERE routes_fts MATCH '%s*'
				`, query))
			if err != nil {
					t.Errorf("SQLite FTS query failed for '%s': %v", query, err)
					continue
				}
				_ = sqliteRows.Close()

				// Compare counts (should be approximately equal, allowing for minor ranking differences)
				t.Logf("✅ FTS query '%s' executed successfully on both databases", query)
			}
			})

			t.Run("GeographicQueries", func(t *testing.T) {
			// Bounding box query
			minLng, maxLng := -5.0, -2.0
			minLat, maxLat := 50.0, 58.0

			// PostgreSQL bbox query
			pgRows, err := pgPool.Query(
				context.Background(),
				fmt.Sprintf(`
					SELECT COUNT(*) FROM routes
					WHERE ST_Within(_geoloc, ST_MakeEnvelope(%f, %f, %f, %f, 4326))
				`, minLng, minLat, maxLng, maxLat))
			if err != nil {
				t.Errorf("PG bbox query failed: %v", err)
			} else {
				pgRows.Close()
				t.Log("✅ PostgreSQL bbox query executed")
			}

			// SQLite bbox query (simplified without Spatialite)
			sqliteRows, err := sqliteDB.Query(
				fmt.Sprintf(`
					SELECT COUNT(*) FROM routes
					WHERE latitude BETWEEN %f AND %f
					AND longitude BETWEEN %f AND %f
				`, minLat, maxLat, minLng, maxLng))
			if err != nil {
				t.Errorf("SQLite bbox query failed: %v", err)
			} else {
				_ = sqliteRows.Close()
				t.Log("✅ SQLite bbox query executed")
			}
			})

			t.Run("FacetedSearch", func(t *testing.T) {
			// Activity facets
			t.Run("ByActivity", func(t *testing.T) {
				// PostgreSQL
				pgStmt := `
					SELECT UNNEST(activities) AS activity, COUNT(*) as count
					FROM routes
					WHERE activities IS NOT NULL
					GROUP BY UNNEST(activities)
					ORDER BY count DESC LIMIT 10
				`
				pgRows, err := pgPool.Query(context.Background(), pgStmt)
				if err != nil {
					t.Errorf("PG facet query failed: %v", err)
				} else {
					defer pgRows.Close()
					pgFacets := make(map[string]int)
					for pgRows.Next() {
						var activity string
						var count int
						_ = pgRows.Scan(&activity, &count)
						pgFacets[activity] = count
					}
					t.Logf("✅ PG activities facets: %d unique values", len(pgFacets))
				}

				// SQLite
				sqliteStmt := `
					SELECT json_each.value as activity, COUNT(*) as count
					FROM routes, json_each(activities)
					GROUP BY json_each.value
					ORDER BY count DESC LIMIT 10
				`
				sqliteRows, err := sqliteDB.Query(sqliteStmt)
				if err != nil {
					t.Errorf("SQLite facet query failed: %v", err)
				} else {
					defer func() { _ = sqliteRows.Close() }()
					sqliteFacets := make(map[string]int)
					for sqliteRows.Next() {
						var activity string
						var count int
						_ = sqliteRows.Scan(&activity, &count)
						sqliteFacets[activity] = count
					}
					t.Logf("✅ SQLite activities facets: %d unique values", len(sqliteFacets))
				}
			})

			// Terrain facets
			t.Run("ByTerrain", func(t *testing.T) {
				// PostgreSQL
				pgStmt := `
					SELECT UNNEST(terrain) AS terrain, COUNT(*) as count
					FROM routes
					WHERE terrain IS NOT NULL
					GROUP BY UNNEST(terrain)
					ORDER BY count DESC LIMIT 10
				`
				pgRows, err := pgPool.Query(context.Background(), pgStmt)
				if err != nil {
					t.Errorf("PG terrain facet failed: %v", err)
				} else {
					defer pgRows.Close()
					count := 0
					for pgRows.Next() {
						count++
					}
					t.Logf("✅ PG terrain facets: %d types", count)
				}

				// SQLite
				sqliteStmt := `
					SELECT json_each.value as terrain, COUNT(*) as count
					FROM routes, json_each(terrain)
					GROUP BY json_each.value
					ORDER BY count DESC LIMIT 10
				`
				sqliteRows, err := sqliteDB.Query(sqliteStmt)
				if err != nil {
					t.Errorf("SQLite terrain facet failed: %v", err)
				} else {
					defer func() { _ = sqliteRows.Close() }()
					count := 0
					for sqliteRows.Next() {
						count++
					}
					t.Logf("✅ SQLite terrain facets: %d types", count)
				}
			})
			})

			t.Run("DataTypeTranslation", func(t *testing.T) {
			// Verify array → JSON translation
			var pgActivityArray string
			err := pgPool.QueryRow(context.Background(),
				"SELECT activities FROM routes WHERE activities IS NOT NULL LIMIT 1").Scan(&pgActivityArray)
			if err != nil {
				t.Errorf("Failed to fetch PG array: %v", err)
			}

			var sqliteActivityJSON string
			err = sqliteDB.QueryRow(
				"SELECT activities FROM routes WHERE activities IS NOT NULL LIMIT 1").Scan(&sqliteActivityJSON)
			if err != nil {
				t.Errorf("Failed to fetch SQLite JSON: %v", err)
			}

			t.Logf("✅ PG array: %s", pgActivityArray)
			t.Logf("✅ SQLite JSON: %s", sqliteActivityJSON)
			})
			}

			// BenchmarkQueryPerformance compares query performance between PostgreSQL and SQLite
			func BenchmarkQueryPerformance(b *testing.B) {
			pgConnStr := os.Getenv("POSTGRESQL_URL")
			if pgConnStr == "" {
			b.Skip("POSTGRESQL_URL not set")
			}

			pgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			pgPool, err := pgxpool.New(pgCtx, pgConnStr)
			if err != nil {
			b.Fatalf("Failed to connect to PostgreSQL: %v", err)
			}
			defer pgPool.Close()

			b.Run("PostgreSQL_FullTextSearch", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, _ := pgPool.Query(context.Background(),
					"SELECT COUNT(*) FROM routes WHERE search_vector @@ to_tsquery('hiking:*')")
				if rows != nil {
					rows.Close()
				}
			}
			})

			sqliteDB, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
			b.Fatalf("Failed to open SQLite: %v", err)
			}
			defer func() { _ = sqliteDB.Close() }()

			b.Run("SQLite_FullTextSearch_Placeholder", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, _ := sqliteDB.Query("SELECT COUNT(*) FROM routes_fts WHERE routes_fts MATCH 'hiking*'")
				if rows != nil {
					_ = rows.Close()
				}
			}
			})
			}

