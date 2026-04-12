package cmds

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/rm-hull/godx"
	"github.com/rm-hull/gps-routes-api/db"
)

func PingDatabase() {

	godx.GitVersion()
	godx.EnvironmentVars()
	godx.UserInfo()

	config := db.ConfigFromEnv()
	ctx := context.Background()

	var sqlDB *sql.DB
	var err error
	var dbType string

	// Check if we should use SQLite:
	// - SQLITE_PATH is set
	// - POSTGRES_HOST is not explicitly set in the environment
	if config.SQLitePath != "" && os.Getenv("POSTGRES_HOST") == "" {
		dbType = "SQLite"
		sqlDB, err = db.NewSQLiteDB(config)
	} else {
		dbType = "PostgreSQL"
		pool, poolErr := db.NewDBPool(ctx, config)
		if poolErr != nil {
			log.Fatalf("Failed to create connection pool: %v", poolErr)
		}
		defer pool.Close()

		// For pinging Postgres via sql.DB interface if needed, but NewDBPool already pings.
		// However, to keep it agnostic for the query below, we can get a sql.DB or just use the pool.
		var result int
		err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
		if err == nil {
			fmt.Printf("Database (%s) ping successful: %v\n", dbType, result)
		}
		return
	}

	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", dbType, err)
	}
	defer sqlDB.Close()

	// Compatible ping query
	var result int
	err = sqlDB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Database (%s) ping successful: %v\n", dbType, result)
}
