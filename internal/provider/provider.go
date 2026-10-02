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

// ErrNotConfigured is returned by optional provider capabilities (such as
// derived index computation) when the provider lacks the required
// configuration, for example OAuth credentials.
var ErrNotConfigured = errors.New("provider is not configured for this operation")

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
	// Item fetches a single item from a collection.
	Item(ctx context.Context, collection, id string) (Observation, error)
}

// IndexProvider is an optional capability implemented by providers that can
// compute derived indices (for example NDVI) server-side.
type IndexProvider interface {
	// SupportsIndex reports whether the provider is configured to compute
	// indices.
	SupportsIndex() bool
	// IndexSeries computes an index over an area and time window.
	IndexSeries(ctx context.Context, req IndexRequest) (IndexSeries, error)
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

// Asset is a downloadable resource attached to an observation.
type Asset struct {
	Name  string   `json:"name"`
	Href  string   `json:"href,omitempty"`
	Type  string   `json:"type,omitempty"`
	Title string   `json:"title,omitempty"`
	Roles []string `json:"roles,omitempty"`
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
	// AssetDetails is populated for single-item fetches, not for searches.
	AssetDetails []Asset        `json:"asset_details,omitempty"`
	Properties   map[string]any `json:"properties,omitempty"`
	ItemURL      string         `json:"item_url,omitempty"`
}

// IndexRequest asks a provider to compute a derived index over an area and a
// time window.
type IndexRequest struct {
	Collection string
	Index      string
	BBox       *geometry.BBox
	Start      *time.Time
	End        *time.Time
	// Interval is an ISO8601 duration for temporal aggregation (for example
	// "P10D").
	Interval string
	// Resolution is the requested ground sample distance in metres.
	Resolution float64
}

// IndexInterval holds aggregated statistics for one time interval.
type IndexInterval struct {
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Mean        *float64  `json:"mean,omitempty"`
	Min         *float64  `json:"min,omitempty"`
	Max         *float64  `json:"max,omitempty"`
	StDev       *float64  `json:"stdev,omitempty"`
	SampleCount *int      `json:"sample_count,omitempty"`
}

// IndexSeries is a derived index aggregated over time.
type IndexSeries struct {
	Index      string          `json:"index"`
	Unit       string          `json:"unit,omitempty"`
	Collection string          `json:"collection,omitempty"`
	Interval   string          `json:"interval,omitempty"`
	Intervals  []IndexInterval `json:"intervals"`
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
