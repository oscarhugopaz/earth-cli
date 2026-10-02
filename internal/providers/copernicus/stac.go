package copernicus

import "encoding/json"

// stacLink is a STAC link object.
type stacLink struct {
	Rel   string `json:"rel"`
	Href  string `json:"href"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

// stacCollection is the subset of a STAC Collection we consume.
type stacCollection struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	License     string      `json:"license"`
	Keywords    []string    `json:"keywords"`
	Extent      *stacExtent `json:"extent"`
	Links       []stacLink  `json:"links"`
}

type stacExtent struct {
	Spatial *struct {
		BBox [][]float64 `json:"bbox"`
	} `json:"spatial"`
	Temporal *struct {
		Interval [][]*string `json:"interval"`
	} `json:"temporal"`
}

type collectionsResponse struct {
	Collections []stacCollection `json:"collections"`
	Links       []stacLink       `json:"links"`
}

// stacFeature is the subset of a STAC Item we consume.
type stacFeature struct {
	ID         string                     `json:"id"`
	Collection string                     `json:"collection"`
	Properties map[string]any             `json:"properties"`
	Geometry   json.RawMessage            `json:"geometry"`
	BBox       []float64                  `json:"bbox"`
	Assets     map[string]json.RawMessage `json:"assets"`
	Links      []stacLink                 `json:"links"`
}

type searchResponse struct {
	Type     string        `json:"type"`
	Features []stacFeature `json:"features"`
	Links    []stacLink    `json:"links"`
}
