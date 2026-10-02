// Package observation implements semantic observations such as "vegetation".
//
// A resolver maps a user-facing concept to a provider collection and returns
// something truthful about what was resolved. It never fabricates derived
// metrics that require processing APIs which are not part of the MVP.
package observation

import (
	"context"
	"sort"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Request is a provider-neutral observation request.
type Request struct {
	BBox        *geometry.BBox
	Start       *time.Time
	End         *time.Time
	PeriodLabel string
	Limit       int
}

// Scene is a lightweight, normalized reference to a source scene.
type Scene struct {
	ID         string     `json:"id"`
	DateTime   *time.Time `json:"datetime,omitempty"`
	CloudCover *float64   `json:"cloud_cover,omitempty"`
	ItemURL    string     `json:"item_url,omitempty"`
}

// Result is what a resolver truthfully knows about an observation.
type Result struct {
	Observation    string                `json:"observation"`
	Description    string                `json:"description,omitempty"`
	Provider       string                `json:"provider"`
	Collection     string                `json:"collection"`
	Source         string                `json:"source"`
	Period         string                `json:"period,omitempty"`
	BBox           []float64             `json:"bbox,omitempty"`
	Scenes         int                   `json:"scenes"`
	CloudScenes    int                   `json:"scenes_with_cloud_cover,omitempty"`
	MeanCloudCover *float64              `json:"mean_cloud_cover,omitempty"`
	BestScene      *Scene                `json:"best_scene,omitempty"`
	Bands          []string              `json:"bands,omitempty"`
	Formula        string                `json:"formula,omitempty"`
	NDVI           *provider.IndexSeries `json:"ndvi,omitempty"`
	Note           string                `json:"note,omitempty"`
	Items          []Scene               `json:"items,omitempty"`
}

// Resolver turns a named observation into a Result using a provider.
type Resolver interface {
	Name() string
	Description() string
	Resolve(ctx context.Context, p provider.Provider, req Request) (*Result, error)
}

// Resolvers returns every registered resolver keyed by name.
func Resolvers() map[string]Resolver {
	all := []Resolver{Vegetation{}}
	out := make(map[string]Resolver, len(all))
	for _, resolver := range all {
		out[resolver.Name()] = resolver
	}
	return out
}

// Names lists registered observation names in stable order.
func Names() []string {
	resolvers := Resolvers()
	names := make([]string, 0, len(resolvers))
	for name := range resolvers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Lookup finds a resolver by name.
func Lookup(name string) (Resolver, bool) {
	resolver, ok := Resolvers()[name]
	return resolver, ok
}
