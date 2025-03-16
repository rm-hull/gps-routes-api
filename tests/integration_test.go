// +build integration

package tests

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rm-hull/gps-routes-api/cmds"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(ctx context.Context) (testcontainers.Container, error) {

    req := testcontainers.ContainerRequest{
        Image:        "postgres:17",
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

    // Set the environment variables expected by db.ConfigFromEnv()
    os.Setenv("DB_HOST", host)
    os.Setenv("DB_PORT", pgPort.Port())
    os.Setenv("DB_USER", "postgres")
    os.Setenv("DB_PASSWORD", "secret")
    os.Setenv("DB_NAME", "testdb")

	cmds.RunMigration("up", "../db/migrations")

	return pgContainer, nil
}

func TestHealthEndpoint(t *testing.T) {

    ctx := context.Background()
	pgContainer, err := setupPostgres(ctx)
	if err != nil {
		t.Logf("Postgres container: %v", err)
	}
	defer pgContainer.Terminate(ctx)

    // Start the HTTP server in a goroutine.
    go cmds.NewHttpServer()

    // Give the server a few seconds to start up.
    time.Sleep(2 * time.Second)

    // Test the /healthz endpoint.
    resp, err := http.Get("http://localhost:8080/healthz")
    if err != nil {
        t.Fatalf("failed to get healthz endpoint: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("unexpected status code, want %d, got %d", http.StatusOK, resp.StatusCode)
    }

    t.Logf("Healthz endpoint returned status %d", resp.StatusCode)
}

// func TestRoutesEndpoint(t *testing.T) {
//     // This is an example placeholder to test your routes endpoints.
//     // You can add any endpoint tests needed following a similar pattern.
//     ctx := context.Background()

//     // Assume the HTTP server is already running from a previous test.
//     url := "http://localhost:8080/your-endpoint" // adjust with your actual endpoint

//     // Wait until the server is accepting connections.
//     var resp *http.Response
//     var err error
//     for i := 0; i < 10; i++ {
//         resp, err = http.Get(url)
//         if err == nil {
//             break
//         }
//         time.Sleep(1 * time.Second)
//     }
//     if err != nil {
//         t.Fatalf("failed to get endpoint %s: %v", url, err)
//     }
//     defer resp.Body.Close()

//     if resp.StatusCode != http.StatusOK {
//         t.Fatalf("unexpected status code for %s, want %d, got %d", url, http.StatusOK, resp.StatusCode)
//     }

//     t.Logf("Endpoint %s returned status %d", url, resp.StatusCode)
// }