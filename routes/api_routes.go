/*
 * GPS Routes API
 *
 * API to retrieve and search GPS Walking Routes
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	model "github.com/rm-hull/gps-routes-api/go"
	svc "github.com/rm-hull/gps-routes-api/services"
)

type RoutesAPI struct {
	Service svc.RoutesService
}

// Get /v1/gps-routes/:objectID
// Retrieve metadata for the specific walking route
func (api *RoutesAPI) FetchRecord(c *gin.Context) {
	objectID := c.Param("objectID")
	if objectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "objectID is required"})
		return
	}
	route, err := api.Service.GetRouteByID(objectID)
	if err != nil {
		if err := c.Error(err); err != nil {
			log.Fatalf("couldnt record error: %v", err)
		}
		return
	}

	if route == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, route)
}

// Post /v1/gps-routes/search
// Search for routes according to various criteria
func (api *RoutesAPI) Search(c *gin.Context) {
	var payload model.SearchRequest
	err := c.ShouldBind(&payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad payload"})
		log.Printf("Error: %v", err)
		return
	}

	matches, err := api.Service.Search(&payload)
	if err != nil {
		if err := c.Error(err); err != nil {
			log.Fatalf("couldnt record error: %v", err)
		}
		return
	}

	c.JSON(200, matches)

}
