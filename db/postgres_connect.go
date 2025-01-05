package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Schema   string
	SSLMode  string
}

func NewDBPool(ctx context.Context, config DBConfig) (*pgxpool.Pool, error) {
	// Construct connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode, config.Schema+",public")

	// Configure the connection pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Set pool settings
	poolConfig.MaxConns = 25                   // Maximum number of connections
	poolConfig.MinConns = 5                    // Minimum number of connections
	poolConfig.MaxConnLifetime = 1 * time.Hour // Max connection lifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Create the pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// Set the schema
	_, err = pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", config.Schema))
	if err != nil {
		return nil, fmt.Errorf("setting schema: %w", err)
	}
	return pool, nil
}

// Same env.vars as per: https://www.postgresql.org/docs/current/libpq-envars.html
func ConfigFromEnv() DBConfig {
	return DBConfig{
		Host:     getEnvOrDefault("PGHOST", "localhost"),
		Port:     getEnvIntOrDefault("PGPORT", 5432),
		User:     getEnvOrDefault("PGUSER", "postgres"),
		Password: getEnvOrDefault("PGPASSWORD", ""),
		DBName:   getEnvOrDefault("PGDATABASE", "postgres"),
		SSLMode:  getEnvOrDefault("PGSSLMODE", "disable"),
		Schema:   getEnvOrDefault("PGSCHEMA", "public"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
