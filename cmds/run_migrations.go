package cmds

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/rm-hull/gps-routes-api/db"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/stdlib"
)

func RunMigration(direction string, migrationsPath string) {
	config := db.ConfigFromEnv()
	ctx := context.Background()
	pool, err := db.NewDBPool(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()

	// Create the migrate instance
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: config.DBName,
		SchemaName:   config.Schema,
	})
	if err != nil {
		log.Fatalf("error creating postgres driver: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf("SET search_path TO %s, public", config.Schema))
	if err != nil {
		log.Fatalf("failed to set search path: %v", err)
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		log.Fatalf("failed convert to convert migrations path to absolute path: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+absPath,
		config.DBName,
		driver,
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	// Run migrations
	if direction == "up" {
		err = m.Up()
	} else {
		err = m.Down()
	}
	if err != nil {
		if err == migrate.ErrNoChange {
			log.Printf("No changes")
		} else {
			log.Fatalf("migrations failed: %v", err)
		}
		return
	}
	log.Printf("Migration applied")
}
