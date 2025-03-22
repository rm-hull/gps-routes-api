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

func setupPostgresContainer(ctx context.Context) (testcontainers.Container, error) {

	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:17-master",
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
	os.Setenv("PGHOST", host)
	os.Setenv("PGPORT", pgPort.Port())
	os.Setenv("PGUSER", "postgres")
	os.Setenv("PGPASSWORD", "secret")
	os.Setenv("PGDATABASE", "testdb")
	os.Setenv("PGSCHEMA", "public")

	cmds.RunMigration("up", "../db/migrations")

	return pgContainer, nil
}
