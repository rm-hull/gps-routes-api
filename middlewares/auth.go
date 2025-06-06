package middlewares

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(notlogged ...string) gin.HandlerFunc {
	apiKey := os.Getenv("GPS_ROUTES_API_KEY")

	return func(c *gin.Context) {

		// Always allow requests for CORS preflight checks
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		for _, path := range notlogged {
			log.Printf("Checking path: %s against notlogged path %s", c.Request.RequestURI, path)
			if path == c.Request.RequestURI {
				c.Next() // Skip authentication for this path
				return
			}
		}

		clientKey := c.GetHeader("X-API-Key")

		if clientKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API Key Required"})
			return
		}

		if clientKey != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			return
		}

		c.Next() // Proceed to the next handler
	}
}
