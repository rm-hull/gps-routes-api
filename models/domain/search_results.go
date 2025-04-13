package domain

type SearchResults struct {
	Total int64 `json:"total"`

	Hits []RouteSummary `json:"hits"`

	Facets map[string]map[string]int64 `json:"facets,omitempty"`

	Attribution []string `json:"attribution"`
}
