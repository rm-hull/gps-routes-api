package tests

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rm-hull/gps-routes-api/cmds"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setEnv(key, value string) error {
	if err := os.Setenv(key, value); err != nil {
		return fmt.Errorf("failed to set environment variable %s: %w", key, err)
	}
	return nil
}

func setupPostgresContainer(ctx context.Context) (testcontainers.Container, error) {

	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/baosystems/postgis:latest",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").
			WithStartupTimeout(30 * time.Second),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, errors.Errorf("failed to start container: %v", err)
	}

	// Get host and port for Postgres
	host, err := pgContainer.Host(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to get container host: %v", err)
	}
	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, errors.Errorf("failed to get mapped port: %v", err)
	}

	fmt.Printf("PostGIS container ID: %s started on %s:%s\n", pgContainer.GetContainerID(), host, pgPort.Port())

	// Set the environment variables expected by db.ConfigFromEnv()
	if err := setEnv("POSTGRES_HOST", host); err != nil {
		return nil, err
	}
	if err := setEnv("POSTGRES_PORT", pgPort.Port()); err != nil {
		return nil, err
	}
	if err := setEnv("POSTGRES_USER", "postgres"); err != nil {
		return nil, err
	}
	if err := setEnv("POSTGRES_PASSWORD", "secret"); err != nil {
		return nil, err
	}
	if err := setEnv("POSTGRES_DB", "testdb"); err != nil {
		return nil, err
	}
	if err := setEnv("POSTGRES_SCHEMA", "public"); err != nil {
		return nil, err
	}

	cmds.RunMigration("up", "../db/migrations")

	return pgContainer, nil
}
