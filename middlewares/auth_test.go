package middlewares

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalAPIKey := os.Getenv("GPS_ROUTES_API_KEY")
	defer func() {
		if err := os.Setenv("GPS_ROUTES_API_KEY", originalAPIKey); err != nil {
			t.Errorf("failed to set GPS_ROUTES_API_KEY environment variable: %v", err)
		}
	}()

	if err := os.Setenv("GPS_ROUTES_API_KEY", "test-api-key"); err != nil {
		t.Errorf("failed to set GPS_ROUTES_API_KEY environment variable: %v", err)
	}
	excluded := []string{"/ping"}

	tests := []struct {
		name           string
		path           string
		method         string
		apiKeyHeader   string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "No API Key",
			path:           "/test",
			method:         http.MethodGet,
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "API Key Required",
		},
		{
			name:           "Invalid API Key",
			path:           "/test",
			method:         http.MethodGet,
			apiKeyHeader:   "invalid-api-key",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid API Key",
		},
		{
			name:           "Valid API Key",
			path:           "/test",
			method:         http.MethodGet,
			apiKeyHeader:   "test-api-key",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Excluded Path",
			path:           "/ping",
			method:         http.MethodGet,
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "OPTIONS Request",
			path:           "/test",
			method:         http.MethodOptions,
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("X-API-Key", tt.apiKeyHeader)

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			middleware := AuthMiddleware(excluded...)
			middleware(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			} else {
				assert.False(t, c.IsAborted())
				c.Abort() // avoid data race
			}
		})
	}

	t.Run("API Key Not Set", func(t *testing.T) {
		originalAPIKey := os.Getenv("GPS_ROUTES_API_KEY")
		defer func() {
			if err := os.Setenv("GPS_ROUTES_API_KEY", originalAPIKey); err != nil {
				t.Errorf("failed to set GPS_ROUTES_API_KEY environment variable: %v", err)
			}
		}()

		if err := os.Unsetenv("GPS_ROUTES_API_KEY"); err != nil {
			t.Errorf("failed to unset GPS_ROUTES_API_KEY environment variable: %v", err)
		}

		assert.Panics(t, func() {
			AuthMiddleware()
		})
	})
}
