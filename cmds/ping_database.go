package cmds

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/map-services/gps-routes-api/db"
	"github.com/rm-hull/godx"
)

func PingDatabase() {

	godx.GitVersion()
	godx.EnvironmentVars()
	godx.UserInfo()

	config := db.ConfigFromEnv()

	ctx := context.Background()
	pool, err := db.NewDBPool(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Example query using the pool
	var now time.Time
	err = pool.QueryRow(ctx, "SELECT NOW()").Scan(&now)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Current time: %v\n", now)
}
