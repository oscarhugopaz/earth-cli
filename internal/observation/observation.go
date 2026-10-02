// Package observation implements semantic observations such as "vegetation",
// "flood" or "burnt-area".
//
// A resolver maps a user-facing concept to a provider collection, resolves the
// relevant scenes and, when the provider is configured, computes a meaningful
// spectral index. It never fabricates derived metrics.
package observation

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/index"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Request is a provider-neutral observation request.
type Request struct {
	BBox        *geometry.BBox
	Start       *time.Time
	End         *time.Time
	PeriodLabel string
	Limit       int
	// Interval is an ISO8601 aggregation duration for derived indices.
	Interval string
	// Resolution is the requested ground sample distance in metres for derived
	// indices.
	Resolution float64
	// Index overrides the resolver's default spectral index (for example
	// "ndwi"). Empty means the resolver's own index.
	Index string
	// DryRun asks the resolver to plan the derived index instead of computing
	// it.
	DryRun bool
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
	Index          *provider.IndexSeries `json:"index,omitempty"`
	IndexPlan      *provider.IndexPlan   `json:"index_plan,omitempty"`
	Note           string                `json:"note,omitempty"`
	Items          []Scene               `json:"items,omitempty"`
}

// Resolver turns a named observation into a Result using a provider.
type Resolver interface {
	Name() string
	Description() string
	Resolve(ctx context.Context, p provider.Provider, req Request) (*Result, error)
}

// Definition is a declarative semantic observation.
type Definition struct {
	// Name is the user-facing identifier (for example "vegetation").
	Name string
	// Description explains what the observation captures.
	Description string
	// Target maps a provider name to the collection and default index used.
	Target map[string]Target
	// Note is shown when no index can be computed (usually missing credentials).
	Note string
}

// Target describes how an observation maps onto a provider collection.
type Target struct {
	Collection string
	Source     string
	// Index is the default spectral index for the observation.
	Index string
	// ProcessingCollection is the data-collection identifier used by the
	// provider's processing API when it differs from the discovery catalogue
	// id (for example STAC uses sentinel-3-sl-2-lst-ntc while Sentinel Hub
	// expects sentinel-3-slstr-l2). Empty means the same as Collection.
	ProcessingCollection string
	// Thermal marks observations that use a temperature product instead of a
	// reflectance index. When set, the resolver uses Evalscript/OutputID/Unit
	// below rather than the spectral index catalog.
	Thermal bool
	// DefaultResolutionM is the ground sample distance used when the request
	// does not specify one. Zero means the provider default (Sentinel-2, 10 m).
	DefaultResolutionM float64
	// Evalscript, OutputID, Unit, Formula describe a custom computation used
	// when Thermal is set.
	Evalscript string
	OutputID   string
	Unit       string
	Formula    string
}

var definitions = []Definition{
	{
		Name:        "vegetation",
		Description: "Vegetation vigor from red and near-infrared reflectance, resolved to Sentinel-2 L2A scenes.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "ndvi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 90d) " +
			"to compute NDVI via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "flood",
		Description: "Surface water extent from green and near-infrared reflectance, resolved to Sentinel-2 L2A scenes.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "mndwi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 30d) " +
			"to compute a water index (MNDWI by default) via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "burnt-area",
		Description: "Burn severity from near-infrared and short-wave infrared reflectance, resolved to Sentinel-2 L2A scenes.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "nbr"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 60d) " +
			"to compute NBR (Normalized Burn Ratio) via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "moisture",
		Description: "Vegetation and soil water content from near-infrared and short-wave infrared reflectance.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "ndmi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 90d) " +
			"to compute NDMI via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "temperature",
		Description: "Land surface temperature from thermal infrared, resolved to Sentinel-3 SLSTR Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           LSTCollection,
				ProcessingCollection: "sentinel-3-slstr-l2",
				Source:               LSTSource,
				Index:                "lst",
				Thermal:              true,
				DefaultResolutionM:   1000,
				Evalscript:           lstEvalscript,
				OutputID:             "lst",
				Unit:                 "°C",
				Formula:              lstFormula,
			},
		},
		Note: "Land surface temperature was not computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 30d) " +
			"to compute LST via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "atmosphere",
		Description: "Atmospheric trace gases (NO2 by default) from Sentinel-5P Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           AtmosphereCollection,
				ProcessingCollection: AtmosphereProcessingCollection,
				Source:               AtmosphereSource,
				Index:                "no2",
				Thermal:              true, // custom evalscript, not the index catalog
				DefaultResolutionM:   3500,
				Evalscript:           AtmosphereEvalscript(),
				OutputID:             "gas",
				Unit:                 "mol/m²",
				Formula:              AtmosphereFormula(),
			},
		},
		Note: "Atmospheric trace gas was not computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 30d) " +
			"to compute it via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "methane",
		Description: "Methane (CH4) column from Sentinel-5P Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           "sentinel-5p-l2-ch4",
				ProcessingCollection: AtmosphereProcessingCollection,
				Source:               AtmosphereSource,
				Index:                "ch4",
				Thermal:              true,
				DefaultResolutionM:   3500,
				Evalscript:           AtmosphereBandEvalscript("CH4"),
				OutputID:             "gas",
				Unit:                 "ppb",
				Formula:              "Sentinel-5P L2 CH4 column (ppb)",
			},
		},
		Note: gasNote("CH4"),
	},
	{
		Name:        "ozone",
		Description: "Ozone (O3) column from Sentinel-5P Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           "sentinel-5p-l2-o3",
				ProcessingCollection: AtmosphereProcessingCollection,
				Source:               AtmosphereSource,
				Index:                "o3",
				Thermal:              true,
				DefaultResolutionM:   3500,
				Evalscript:           AtmosphereBandEvalscript("O3"),
				OutputID:             "gas",
				Unit:                 "DU",
				Formula:              "Sentinel-5P L2 O3 column (DU)",
			},
		},
		Note: gasNote("O3"),
	},
	{
		Name:        "carbon-monoxide",
		Description: "Carbon monoxide (CO) column from Sentinel-5P Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           "sentinel-5p-l2-co",
				ProcessingCollection: AtmosphereProcessingCollection,
				Source:               AtmosphereSource,
				Index:                "co",
				Thermal:              true,
				DefaultResolutionM:   3500,
				Evalscript:           AtmosphereBandEvalscript("CO"),
				OutputID:             "gas",
				Unit:                 "mol/m²",
				Formula:              "Sentinel-5P L2 CO column",
			},
		},
		Note: gasNote("CO"),
	},
	{
		Name:        "sulfur-dioxide",
		Description: "Sulfur dioxide (SO2) column from Sentinel-5P Level-2.",
		Target: map[string]Target{
			"copernicus": {
				Collection:           "sentinel-5p-l2-so2",
				ProcessingCollection: AtmosphereProcessingCollection,
				Source:               AtmosphereSource,
				Index:                "so2",
				Thermal:              true,
				DefaultResolutionM:   3500,
				Evalscript:           AtmosphereBandEvalscript("SO2"),
				OutputID:             "gas",
				Unit:                 "mol/m²",
				Formula:              "Sentinel-5P L2 SO2 column",
			},
		},
		Note: gasNote("SO2"),
	},
	{
		Name:        "snow",
		Description: "Snow and ice extent from green and short-wave infrared reflectance.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "ndsi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 30d) " +
			"to compute NDSI via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "urban",
		Description: "Built-up and impervious surfaces from short-wave infrared and near-infrared reflectance.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "ndbi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 90d) " +
			"to compute NDBI via the Sentinel Hub Statistical API.",
	},
	{
		Name:        "crop",
		Description: "Cropland vigor from green and near-infrared reflectance, resolved to Sentinel-2 L2A scenes.",
		Target: map[string]Target{
			"copernicus": {Collection: "sentinel-2-l2a", Source: "Sentinel-2 Level-2A", Index: "gndvi"},
		},
		Note: "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
			"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 90d) " +
			"to compute GNDVI via the Sentinel Hub Statistical API.",
	},
}

// fireAliases map alternative names onto a registered observation.
var fireAliases = map[string]string{
	"fire":        "burnt-area",
	"burn":        "burnt-area",
	"burn-area":   "burnt-area",
	"water":       "flood",
	"ndvi":        "vegetation",
	"ice":         "snow",
	"built-up":    "urban",
	"cropland":    "crop",
	"agriculture": "crop",
}

const dryRunNote = "Dry run: no index was computed and no processing units were spent. " +
	"Remove --dry-run to run the request against the Sentinel Hub Statistical API."

type resolver struct {
	def Definition
}

// Resolvers returns every registered resolver keyed by name and alias.
func Resolvers() map[string]Resolver {
	out := make(map[string]Resolver, len(definitions)+len(fireAliases))
	for _, def := range definitions {
		out[def.Name] = resolver{def: def}
	}
	for alias, canonical := range fireAliases {
		if canonicalDef, ok := definitionByName(canonical); ok {
			out[alias] = resolver{def: canonicalDef}
		}
	}
	return out
}

// Names lists unique canonical observation names in stable order.
func Names() []string {
	names := make([]string, 0, len(definitions))
	for _, def := range definitions {
		names = append(names, def.Name)
	}
	sort.Strings(names)
	return names
}

// Lookup finds a resolver by canonical name or alias.
func Lookup(name string) (Resolver, bool) {
	res, ok := Resolvers()[strings.ToLower(strings.TrimSpace(name))]
	return res, ok
}

// CanonicalName resolves an alias to its canonical observation name.
func CanonicalName(name string) (string, bool) {
	res, ok := Lookup(name)
	if !ok {
		return "", false
	}
	return res.Name(), true
}

// TargetFor returns the provider mapping for a canonical observation name.
func TargetFor(canonical, providerName string) (Target, bool) {
	def, ok := definitionByName(canonical)
	if !ok {
		return Target{}, false
	}
	target, ok := def.Target[providerName]
	return target, ok
}

// UnknownError builds an actionable error for an unknown observation.
func UnknownError(name string) error {
	return fmt.Errorf("unknown observation %q\n\nAvailable observations:\n  %s",
		name, strings.Join(Names(), "\n  "))
}

func definitionByName(name string) (Definition, bool) {
	for _, def := range definitions {
		if def.Name == name {
			return def, true
		}
	}
	return Definition{}, false
}

func (r resolver) Name() string        { return r.def.Name }
func (r resolver) Description() string { return r.def.Description }

// Resolve implements Resolver.
func (r resolver) Resolve(ctx context.Context, p provider.Provider, req Request) (*Result, error) {
	mapping, ok := r.def.Target[p.Name()]
	if !ok {
		return nil, fmt.Errorf("observation %q is not available from provider %q", r.def.Name, p.Name())
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	scenes, err := p.Search(ctx, provider.SearchRequest{
		Collection: mapping.Collection,
		BBox:       req.BBox,
		Start:      req.Start,
		End:        req.End,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	result := &Result{
		Observation: r.def.Name,
		Description: r.def.Description,
		Provider:    p.Name(),
		Collection:  mapping.Collection,
		Source:      mapping.Source,
		Period:      req.PeriodLabel,
		Note:        r.def.Note,
		Scenes:      len(scenes),
	}
	if req.BBox != nil {
		result.BBox = req.BBox.Slice()
	}

	var cloudSum float64
	var cloudCount int
	for i := range scenes {
		scene := scenes[i]
		result.Items = append(result.Items, Scene{
			ID:         scene.ID,
			DateTime:   scene.DateTime,
			CloudCover: scene.CloudCover,
			ItemURL:    scene.ItemURL,
		})
		if scene.CloudCover != nil {
			cloudSum += *scene.CloudCover
			cloudCount++
		}
	}
	result.CloudScenes = cloudCount
	if cloudCount > 0 {
		mean := cloudSum / float64(cloudCount)
		result.MeanCloudCover = &mean
	}

	if best := selectBestScene(scenes); best != nil {
		result.BestScene = &Scene{
			ID:         best.ID,
			DateTime:   best.DateTime,
			CloudCover: best.CloudCover,
			ItemURL:    best.ItemURL,
		}
	}

	// Compute the index only when the provider supports it and we have both an
	// area and an explicit time window. Never fabricate a value.
	if result.BBox != nil && req.Start != nil && req.End != nil {
		if indexer, ok := p.(provider.IndexProvider); ok && indexer.SupportsIndex() {
			indexName := strings.TrimSpace(req.Index)
			if indexName == "" {
				indexName = mapping.Index
			}
			if !mapping.Thermal {
				if _, known := index.Lookup(indexName); !known {
					return nil, index.UnknownError(indexName)
				}
			}
			indexReq := provider.IndexRequest{
				Collection: mapping.Collection,
				Index:      indexName,
				BBox:       req.BBox,
				Start:      req.Start,
				End:        req.End,
				Interval:   req.Interval,
				Resolution: req.Resolution,
			}
			if indexReq.Resolution <= 0 && mapping.DefaultResolutionM > 0 {
				indexReq.Resolution = mapping.DefaultResolutionM
			}
			if mapping.Thermal {
				// The processing API may use a different collection id than the
				// discovery catalogue.
				if mapping.ProcessingCollection != "" {
					indexReq.Collection = mapping.ProcessingCollection
				}
				indexReq.Evalscript = mapping.Evalscript
				indexReq.OutputID = mapping.OutputID
				indexReq.Unit = mapping.Unit
				indexReq.Formula = mapping.Formula
			}
			if req.DryRun {
				plan := indexer.PlanIndex(indexReq)
				if plan.Index == "" {
					return nil, index.UnknownError(indexName)
				}
				result.IndexPlan = &plan
				result.Note = dryRunNote
				return result, nil
			}
			series, err := indexer.IndexSeries(ctx, indexReq)
			if err != nil {
				return nil, err
			}
			result.Index = &series
			result.Formula = series.Formula
			if mapping.Thermal {
				result.Bands = nil
				result.Note = thermalNote(series.Title, mapping.Source)
			} else {
				result.Bands = append([]string(nil), indexBands(series.Index)...)
				result.Note = indexNote(series.Title)
			}
		}
	}

	return result, nil
}

func indexNote(title string) string {
	if title == "" {
		title = "The index"
	}
	return fmt.Sprintf("%s computed from Sentinel-2 L2A via the Sentinel Hub Statistical API; "+
		"clouds, shadows and snow are masked using the Scene Classification Layer.", title)
}

func thermalNote(title, source string) string {
	if title == "" {
		title = "Land surface temperature"
	}
	if source == "" {
		source = "a satellite product"
	}
	return fmt.Sprintf("%s computed from %s via the Sentinel Hub Statistical API.", title, source)
}

// gasNote builds the note shown when a trace gas could not be computed.
func gasNote(band string) string {
	return fmt.Sprintf("%s was not computed. Set EARTH_COPERNICUS_CLIENT_ID and "+
		"EARTH_COPERNICUS_CLIENT_SECRET and provide a time window (for example --since 30d) "+
		"to compute it via the Sentinel Hub Statistical API.", band)
}

// indexBands returns the human-facing bands for an index name.
func indexBands(name string) []string {
	if def, ok := index.Lookup(name); ok {
		return append([]string(nil), def.Bands...)
	}
	return nil
}

// selectBestScene prefers the least cloudy scene and breaks ties by recency.
func selectBestScene(scenes []provider.Observation) *provider.Observation {
	var best *provider.Observation
	for i := range scenes {
		candidate := &scenes[i]
		if best == nil || betterScene(candidate, best) {
			best = candidate
		}
	}
	return best
}

func betterScene(candidate, current *provider.Observation) bool {
	if candidate.CloudCover != nil && current.CloudCover != nil {
		if *candidate.CloudCover != *current.CloudCover {
			return *candidate.CloudCover < *current.CloudCover
		}
	} else if candidate.CloudCover != nil {
		return true
	} else if current.CloudCover != nil {
		return false
	}

	if candidate.DateTime != nil && current.DateTime != nil {
		return candidate.DateTime.After(*current.DateTime)
	}
	return false
}
