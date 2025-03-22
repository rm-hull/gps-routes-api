package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/rm-hull/gps-routes-api/cmds"
)

func TestHealthEndpoint(t *testing.T) {

	// Start the HTTP server in a goroutine.
	go cmds.NewHttpServer()

	// Give the server a few seconds to start up.
	time.Sleep(5 * time.Second)

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

func TestMetricsEndpoint(t *testing.T) {

	// Test the /healthz endpoint.
	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Fatalf("failed to get healthz endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	t.Logf("Healthz endpoint returned status %d", resp.StatusCode)
}
