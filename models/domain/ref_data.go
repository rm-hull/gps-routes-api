package domain

type RefData struct {
	Facets      Facets   `json:"facets,omitempty"`
	Attribution []string `json:"attribution"`
}
