package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, ginErr := range c.Errors {
				log.Printf("Error: %v", ginErr.Err)
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "An unexpected error occurred",
			})
		}
	}
}
