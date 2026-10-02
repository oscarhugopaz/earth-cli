package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
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

	// resx/resy are expressed in the units of the requested CRS. Our bounds
	// use EPSG:4326 (degrees), so convert metres to degrees at the bbox centre.
	resDeg := metresToDegrees(resolution, req.BBox)

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
			"resx":                resDeg,
			"resy":                resDeg,
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
	Stats indexStats `json:"stats"`
}

// metresToDegrees converts a ground sample distance in metres to degrees at
// the bbox centre, approximating the WGS84 surface. Sentinel Hub expects
// resx/resy in the units of the requested CRS.
func metresToDegrees(metres float64, bbox *geometry.BBox) float64 {
	const metresPerDegree = 111320.0 // at the equator; adjusted for latitude
	if bbox == nil || metres <= 0 {
		return metres / metresPerDegree
	}
	centreLat := (bbox.MinLat + bbox.MaxLat) / 2
	cosLat := math.Cos(centreLat * math.Pi / 180)
	if cosLat < 0.01 { // degrade gracefully near the poles
		cosLat = 0.01
	}
	return metres / (metresPerDegree * cosLat)
}

// indexStats is tolerant of numbers encoded as JSON strings and of the
// non-numeric placeholders Sentinel Hub uses ("NaN", "Infinity") when an
// interval has no valid samples.
type indexStats struct {
	Mean        *float64
	Min         *float64
	Max         *float64
	StDev       *float64
	SampleCount *int
}

func (s *indexStats) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.Mean = flexFloat(raw["mean"])
	s.Min = flexFloat(raw["min"])
	s.Max = flexFloat(raw["max"])
	s.StDev = flexFloat(raw["stDev"])
	s.SampleCount = flexInt(raw["sampleCount"])
	return nil
}

func flexFloat(raw json.RawMessage) *float64 {
	if len(raw) == 0 {
		return nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return nil
		}
		return &number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	return &value
}

func flexInt(raw json.RawMessage) *int {
	value := flexFloat(raw)
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

// statisticsFor extracts the first band's stats for the named output.
func statisticsFor(outputs map[string]statisticsOutput, index string) (indexStats, bool) {
	output, ok := outputs[index]
	if !ok {
		return indexStats{}, false
	}
	for _, band := range output.Bands {
		return band.Stats, true
	}
	return indexStats{}, false
}
