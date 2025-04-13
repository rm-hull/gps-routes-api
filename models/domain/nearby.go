package domain

type Nearby struct {

	// The object ID identifies the GPS route (typically this is the MD5 hash of the route reference)
	ObjectID string `json:"objectID"`

	// A human friendly unique identifier for the route.
	Ref string `json:"ref"`

	// The route title
	Description string `json:"description"`
}
