package typesense

// Collection : typesense collection
type Collection struct {
	Name   string `json:"name"`
	Fields []struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Facet bool   `json:"facet,omitempty"`
	} `json:"fields"`
	DefaultSortingField string `json:"default_sorting_field"`
}
