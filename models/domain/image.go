package domain

type Image struct {

	// A URL referencing an image
	Src string `json:"src"`

	// Description, typically containing some attribution for the image
	Title string `json:"title,omitempty"`

	// Appropriate image caption
	Caption string `json:"caption,omitempty"`
}
