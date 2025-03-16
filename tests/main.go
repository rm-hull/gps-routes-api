package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup shared container
	ctx := context.Background()
	postgres, err := setupPostgresContainer(ctx)
	if err != nil {
		fmt.Printf("Failed to start container: %s\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Clean up
	if err := postgres.Terminate(ctx); err != nil {
		fmt.Printf("Failed to terminate container: %s\n", err)
	}

	os.Exit(code)
}
