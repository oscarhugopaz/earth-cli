// Package provider defines the provider-oriented domain model used by the
// Earth Engine. Concrete catalogs (Copernicus, NASA, Planetary Computer, ...)
// implement the Provider interface so the CLI never talks to one catalog
// directly.
package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
)

// ErrNotFound is returned when a collection or item does not exist.
var ErrNotFound = errors.New("not found")

// Provider is a source of Earth observation data.
type Provider interface {
	// Name is the stable identifier used on the command line (for example
	// "copernicus").
	Name() string
	// Type describes the access protocol (for example "stac").
	Type() string
	// Status reports availability (for example "available").
	Status() string
	// Description is a short human-readable summary.
	Description() string
	// Collections lists discoverable collections.
	Collections(ctx context.Context, limit int) ([]Collection, error)
	// Collection fetches a single collection by id.
	Collection(ctx context.Context, id string) (Collection, error)
	// Search finds observations matching req.
	Search(ctx context.Context, req SearchRequest) ([]Observation, error)
}

// SearchRequest is a provider-neutral search.
type SearchRequest struct {
	Collection string
	BBox       *geometry.BBox
	Start      *time.Time
	End        *time.Time
	Limit      int
}

// Collection is a normalized STAC collection.
type Collection struct {
	ID            string   `json:"id"`
	Provider      string   `json:"provider"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	License       string   `json:"license,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	Extent        *Extent  `json:"extent,omitempty"`
	ItemURL       string   `json:"item_url,omitempty"`
	QueryablesURL string   `json:"queryables_url,omitempty"`
}

// Extent describes the spatial and temporal coverage of a collection.
type Extent struct {
	Spatial  [][]float64 `json:"spatial_bbox,omitempty"`
	Temporal [][]*string `json:"temporal_interval,omitempty"`
}

// Observation is a normalized STAC item.
type Observation struct {
	ID         string          `json:"id"`
	Collection string          `json:"collection"`
	Provider   string          `json:"provider"`
	DateTime   *time.Time      `json:"datetime,omitempty"`
	BBox       []float64       `json:"bbox,omitempty"`
	Geometry   json.RawMessage `json:"geometry,omitempty"`
	CloudCover *float64        `json:"cloud_cover,omitempty"`
	Assets     []string        `json:"assets,omitempty"`
	Properties map[string]any  `json:"properties,omitempty"`
	ItemURL    string          `json:"item_url,omitempty"`
}

// ResponseError is returned when a provider responds with an unexpected HTTP
// status. It is intentionally actionable.
type ResponseError struct {
	Provider string
	Method   string
	URL      string
	Status   string
	Code     int
	Body     string
}

func (e *ResponseError) Error() string {
	msg := fmt.Sprintf("%s STAC request failed: HTTP %s", e.Provider, e.Status)
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}
