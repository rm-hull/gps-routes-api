package domain

type SearchResults struct {
	Total int64 `json:"total"`

	Hits []RouteSummary `json:"hits"`

	Facets Facets `json:"facets,omitempty"`

	Attribution []string `json:"attribution"`
}
