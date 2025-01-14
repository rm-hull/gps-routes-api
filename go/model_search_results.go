/*
 * GPS Routes API
 *
 * API to retrieve and search GPS Walking Routes
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type SearchResults struct {
	Total int64 `json:"total"`

	Hits []RouteSummary `json:"hits"`

	Facets map[string]map[string]int64 `json:"facets,omitempty"`

	Attribution []string `json:"attribution"`
}
