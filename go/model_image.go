/*
 * GPS Routes API
 *
 * API to retrieve and search GPS Walking Routes
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

type Image struct {

	// A URL referencing an image
	Src string `json:"src"`

	// Description, typically containing some attribution for the image
	Title string `json:"title,omitempty"`

	// Appropriate image caption
	Caption string `json:"caption,omitempty"`
}