package observation

import (
	"context"
	"fmt"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Vegetation resolves the source scenes needed for vegetation monitoring.
//
// It maps the semantic concept to a provider collection, fetches candidate
// scenes and reports scene statistics. NDVI itself is intentionally not
// computed here: that requires the authenticated Sentinel Hub
// Processing/Statistical APIs or local raster processing, which are out of
// scope for the MVP.
type Vegetation struct{}

// target describes how "vegetation" maps onto a concrete provider collection.
type target struct {
	Collection string
	Source     string
	Bands      []string
	Formula    string
}

var vegetationTargets = map[string]target{
	"copernicus": {
		Collection: "sentinel-2-l2a",
		Source:     "Sentinel-2 Level-2A",
		Bands:      []string{"B04", "B08"},
		Formula:    "NDVI = (B08 - B04) / (B08 + B04)",
	},
}

const vegetationNote = "NDVI was not computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
	"EARTH_COPERNICUS_CLIENT_SECRET (a Copernicus Sentinel Hub OAuth client) and provide a time " +
	"window (for example --since 90d) to compute NDVI via the Sentinel Hub Statistical API. " +
	"Without them, earth only resolves the source scenes and does not fabricate a value."

const vegetationIndexNote = "NDVI computed from Sentinel-2 L2A via the Sentinel Hub Statistical API; " +
	"clouds, shadows and snow are masked using the Scene Classification Layer. NDVI = (B08 - B04) / (B08 + B04)."

const vegetationDryRunNote = "Dry run: no NDVI was computed and no processing units were spent. " +
	"Remove --dry-run to run the request against the Sentinel Hub Statistical API."

func (Vegetation) Name() string { return "vegetation" }

func (Vegetation) Description() string {
	return "Vegetation vigor from red and near-infrared reflectance, resolved to Sentinel-2 L2A scenes."
}

// Resolve implements Resolver.
func (v Vegetation) Resolve(ctx context.Context, p provider.Provider, req Request) (*Result, error) {
	mapping, ok := vegetationTargets[p.Name()]
	if !ok {
		return nil, fmt.Errorf("observation %q is not available from provider %q", v.Name(), p.Name())
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
		Observation: v.Name(),
		Description: v.Description(),
		Provider:    p.Name(),
		Collection:  mapping.Collection,
		Source:      mapping.Source,
		Period:      req.PeriodLabel,
		Bands:       append([]string(nil), mapping.Bands...),
		Formula:     mapping.Formula,
		Note:        vegetationNote,
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

	// Compute NDVI only when the provider supports it and we have both an area
	// and an explicit time window. Never fabricate a value.
	if result.BBox != nil && req.Start != nil && req.End != nil {
		if indexer, ok := p.(provider.IndexProvider); ok && indexer.SupportsIndex() {
			indexReq := provider.IndexRequest{
				Collection: mapping.Collection,
				Index:      "ndvi",
				BBox:       req.BBox,
				Start:      req.Start,
				End:        req.End,
				Interval:   req.Interval,
				Resolution: req.Resolution,
			}
			if req.DryRun {
				plan := indexer.PlanIndex(indexReq)
				result.NDVIPlan = &plan
				result.Note = vegetationDryRunNote
				return result, nil
			}
			series, err := indexer.IndexSeries(ctx, indexReq)
			if err != nil {
				return nil, err
			}
			result.NDVI = &series
			result.Note = vegetationIndexNote
		}
	}

	return result, nil
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
