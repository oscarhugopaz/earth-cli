package observation

import (
	"context"
	"fmt"
	"strings"

	"github.com/oscarhugopaz/earth-cli/internal/index"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Vegetation resolves the source scenes needed for vegetation monitoring.
//
// It maps the semantic concept to a provider collection, fetches candidate
// scenes, reports scene statistics and, when the provider is configured with
// credentials, computes a spectral index (NDVI by default) over the area and
// time window. It never fabricates an index value.
type Vegetation struct{}

// target describes how an observation maps onto a concrete provider collection.
type target struct {
	Collection string
	Source     string
	// Index is the default spectral index for the observation.
	Index string
}

var vegetationTargets = map[string]target{
	"copernicus": {
		Collection: "sentinel-2-l2a",
		Source:     "Sentinel-2 Level-2A",
		Index:      "ndvi",
	},
}

const noIndexNote = "No spectral index was computed. Set EARTH_COPERNICUS_CLIENT_ID and " +
	"EARTH_COPERNICUS_CLIENT_SECRET (a Copernicus Sentinel Hub OAuth client) and provide a time " +
	"window (for example --since 90d) to compute an index via the Sentinel Hub Statistical API. " +
	"Without them, earth only resolves the source scenes and does not fabricate a value."

const dryRunNote = "Dry run: no index was computed and no processing units were spent. " +
	"Remove --dry-run to run the request against the Sentinel Hub Statistical API."

func indexNote(title string) string {
	if title == "" {
		title = "The index"
	}
	return fmt.Sprintf("%s computed from Sentinel-2 L2A via the Sentinel Hub Statistical API; "+
		"clouds, shadows and snow are masked using the Scene Classification Layer.", title)
}

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
		Note:        noIndexNote,
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
			indexReq := provider.IndexRequest{
				Collection: mapping.Collection,
				Index:      indexName,
				BBox:       req.BBox,
				Start:      req.Start,
				End:        req.End,
				Interval:   req.Interval,
				Resolution: req.Resolution,
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
			result.Bands = append([]string(nil), indexBands(series.Index)...)
			result.Note = indexNote(series.Title)
		}
	}

	return result, nil
}

// indexBands returns the human-facing bands for an index name.
func indexBands(name string) []string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ndvi", "savi":
		return []string{"B04", "B08"}
	case "ndwi":
		return []string{"B03", "B08"}
	case "mndwi":
		return []string{"B03", "B11"}
	case "ndmi":
		return []string{"B08", "B11"}
	case "nbr":
		return []string{"B08", "B12"}
	case "ndbi":
		return []string{"B11", "B08"}
	case "evi":
		return []string{"B02", "B04", "B08"}
	case "ndre":
		return []string{"B05", "B08"}
	default:
		return nil
	}
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
