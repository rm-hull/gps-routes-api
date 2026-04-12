package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteDB opens a connection to a SQLite database and sets standard pragmas
func NewSQLiteDB(config DBConfig) (*sql.DB, error) {
	if config.SQLitePath == "" {
		return nil, fmt.Errorf("sqlite path is empty")
	}

	// Open connection
	db, err := sql.Open("sqlite3", config.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database at %s: %w", config.SQLitePath, err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Set Pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	// Load mod_spatialite extension
	// Note: some environments might need different names, but 'mod_spatialite' is standard.
	// If it fails, log a warning but don't fail (unless strictly required).
	_, err = db.Exec("SELECT load_extension('mod_spatialite')")
	if err != nil {
		log.Printf("Warning: failed to load mod_spatialite extension: %v", err)
	}

	return db, nil
}
