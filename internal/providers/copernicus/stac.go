package copernicus

import "encoding/json"

// stacLink is a STAC link object. Method and Body are used by pagination
// links, which may be POST requests carrying a token.
type stacLink struct {
	Rel    string          `json:"rel"`
	Href   string          `json:"href"`
	Type   string          `json:"type,omitempty"`
	Title  string          `json:"title,omitempty"`
	Method string          `json:"method,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
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

// stacAsset is a STAC asset object.
type stacAsset struct {
	Href  string   `json:"href"`
	Type  string   `json:"type,omitempty"`
	Title string   `json:"title,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

// stacFeature is the subset of a STAC Item we consume.
type stacFeature struct {
	ID         string               `json:"id"`
	Collection string               `json:"collection"`
	Properties map[string]any       `json:"properties"`
	Geometry   json.RawMessage      `json:"geometry"`
	BBox       []float64            `json:"bbox"`
	Assets     map[string]stacAsset `json:"assets"`
	Links      []stacLink           `json:"links"`
}

type searchResponse struct {
	Type     string        `json:"type"`
	Features []stacFeature `json:"features"`
	Links    []stacLink    `json:"links"`
}
