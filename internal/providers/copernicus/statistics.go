package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// ndviEvalscript computes NDVI from Sentinel-2 red (B04) and near-infrared
// (B08) bands, masking cloud, shadow and snow pixels via the Scene
// Classification Layer (SCL).
const ndviEvalscript = `//VERSION=3
function setup() {
  return {
    input: [{ bands: ["B04", "B08", "SCL", "dataMask"] }],
    output: [
      { id: "ndvi", bands: 1, sampleType: "FLOAT32" },
      { id: "dataMask", bands: 1 }
    ]
  };
}
function evaluatePixel(sample) {
  var valid = sample.dataMask === 1 && [3, 8, 9, 10, 11].indexOf(sample.SCL) === -1;
  var ndvi = (sample.B08 - sample.B04) / (sample.B08 + sample.B04);
  return { ndvi: [ndvi], dataMask: [valid ? 1 : 0] };
}`

// SupportsIndex reports whether OAuth credentials are configured.
func (p *Provider) SupportsIndex() bool { return p.auth != nil }

// IndexSeries computes an index over an area and time window using the
// Sentinel Hub Statistical API.
func (p *Provider) IndexSeries(ctx context.Context, req provider.IndexRequest) (provider.IndexSeries, error) {
	if p.auth == nil {
		return provider.IndexSeries{}, provider.ErrNotConfigured
	}

	index := strings.ToLower(strings.TrimSpace(req.Index))
	if index == "" {
		index = "ndvi"
	}
	if index != "ndvi" {
		return provider.IndexSeries{}, fmt.Errorf("unsupported index %q: only ndvi is implemented", req.Index)
	}
	if req.BBox == nil || req.Start == nil || req.End == nil {
		return provider.IndexSeries{}, fmt.Errorf("index %s requires an area and a time window", index)
	}

	collection := strings.TrimSpace(req.Collection)
	if collection == "" {
		collection = "sentinel-2-l2a"
	}
	interval := strings.TrimSpace(req.Interval)
	if interval == "" {
		interval = "P10D"
	}
	resolution := req.Resolution
	if resolution <= 0 {
		resolution = 10
	}

	body := map[string]any{
		"input": map[string]any{
			"bounds": map[string]any{
				"bbox": req.BBox.Slice(),
				"properties": map[string]any{
					"crs": "http://www.opengis.net/def/crs/EPSG/0/4326",
				},
			},
			"data": []map[string]any{{"type": collection}},
		},
		"aggregation": map[string]any{
			"timeRange": map[string]string{
				"from": req.Start.UTC().Format(time.RFC3339),
				"to":   req.End.UTC().Format(time.RFC3339),
			},
			"aggregationInterval": map[string]string{"of": interval},
			"evalscript":          ndviEvalscript,
			"resx":                resolution,
			"resy":                resolution,
		},
		"calculations": map[string]any{
			"default": map[string]any{
				"statistics": map[string]any{"default": map[string]any{}},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return provider.IndexSeries{}, fmt.Errorf("encode Copernicus statistical request: %w", err)
	}

	token, err := p.auth.accessToken(ctx)
	if err != nil {
		return provider.IndexSeries{}, err
	}

	var response statisticsResponse
	if err := p.request(ctx, http.MethodPost, p.statisticsURL, payload, token, &response); err != nil {
		return provider.IndexSeries{}, err
	}

	series := provider.IndexSeries{
		Index:      index,
		Unit:       "index",
		Collection: collection,
		Interval:   interval,
		Intervals:  make([]provider.IndexInterval, 0, len(response.Data)),
	}
	for _, bucket := range response.Data {
		entry := provider.IndexInterval{From: bucket.Interval.From, To: bucket.Interval.To}
		if stats, ok := statisticsFor(bucket.Outputs, index); ok {
			entry.Mean = stats.Mean
			entry.Min = stats.Min
			entry.Max = stats.Max
			entry.StDev = stats.StDev
			entry.SampleCount = stats.SampleCount
		}
		series.Intervals = append(series.Intervals, entry)
	}

	return series, nil
}

type statisticsResponse struct {
	Status string             `json:"status"`
	Data   []statisticsBucket `json:"data"`
}

type statisticsBucket struct {
	Interval struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"interval"`
	Outputs map[string]statisticsOutput `json:"outputs"`
}

type statisticsOutput struct {
	Bands map[string]statisticsBand `json:"bands"`
}

type statisticsBand struct {
	Stats statisticsStats `json:"stats"`
}

type statisticsStats struct {
	Mean        *float64 `json:"mean"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	StDev       *float64 `json:"stDev"`
	SampleCount *int     `json:"sampleCount"`
}

// statisticsFor extracts the first band's stats for the named output.
func statisticsFor(outputs map[string]statisticsOutput, index string) (statisticsStats, bool) {
	output, ok := outputs[index]
	if !ok {
		return statisticsStats{}, false
	}
	for _, band := range output.Bands {
		return band.Stats, true
	}
	return statisticsStats{}, false
}
