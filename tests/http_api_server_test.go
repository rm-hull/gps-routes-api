package tests

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rm-hull/gps-routes-api/cmds"
)

func TestHealthEndpoint(t *testing.T) {
	originalAPIKey := os.Getenv("GPS_ROUTES_API_KEY")
	defer func() {
		os.Setenv("GPS_ROUTES_API_KEY", originalAPIKey)
	}()

	os.Setenv("GPS_ROUTES_API_KEY", "test-api-key")

	// Start the HTTP server in a goroutine.
	go cmds.NewHttpApiServer(8080)

	// Give the server a few seconds to start up.
	time.Sleep(5 * time.Second)

	resp, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		t.Fatalf("failed to get healthz endpoint: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	t.Logf("Healthz endpoint returned status %d", resp.StatusCode)
}

func TestMetricsEndpoint(t *testing.T) {
	originalAPIKey := os.Getenv("GPS_ROUTES_API_KEY")
	defer func() {
		os.Setenv("GPS_ROUTES_API_KEY", originalAPIKey)
	}()

	os.Setenv("GPS_ROUTES_API_KEY", "test-api-key")

	// Test the /healthz endpoint.
	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Fatalf("failed to get metrics endpoint: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	t.Logf("Metrics endpoint returned status %d", resp.StatusCode)
}
