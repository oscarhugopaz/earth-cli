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

const vegetationNote = "earth resolves the source scenes but does not compute NDVI in this release: " +
	"NDVI requires the Sentinel Hub Processing/Statistical APIs or local raster processing, which are not part of the MVP."

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
