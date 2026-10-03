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
	"github.com/oscarhugopaz/earth-cli/internal/index"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// SupportsIndex reports whether OAuth credentials are configured.
func (p *Provider) SupportsIndex() bool { return p.auth != nil && !p.indicesDisabled }

// PlanIndex returns what an index request would do and a PU estimate, without
// contacting Sentinel Hub.
func (p *Provider) PlanIndex(req provider.IndexRequest) provider.IndexPlan {
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

	name := strings.ToLower(strings.TrimSpace(req.Index))
	if name == "" {
		name = "ndvi"
	}

	// A custom evalscript carries its own index definition (for example land
	// surface temperature), so it does not need a catalogue entry.
	custom := strings.TrimSpace(req.Evalscript) != ""

	var def index.Definition
	if !custom {
		var ok bool
		def, ok = index.Lookup(name)
		if !ok {
			// Signal an unknown index by leaving Index empty; callers surface a
			// helpful error instead of planning a bogus request.
			return provider.IndexPlan{}
		}
	}

	title := def.Title
	if req.Title != "" {
		title = req.Title
	}
	formula := def.Formula
	if req.Formula != "" {
		formula = req.Formula
	}

	var bands []string
	if custom {
		bands = nil // the output id ("lst") is not a band; keep it empty
	} else {
		bands = indexBands(def)
	}

	plan := provider.IndexPlan{
		Collection:  collection,
		Index:       name,
		Interval:    interval,
		ResolutionM: resolution,
		Bands:       bands,
		Formula:     formula,
		Title:       title,
		Description: def.Description,
	}
	if req.BBox != nil {
		plan.BBox = req.BBox.Slice()
	}
	plan.Start = req.Start
	plan.End = req.End

	// The number of acquisitions is unknown without querying the catalogue, so
	// the estimate assumes a Sentinel-2 revisit of about 5 days.
	if req.Start != nil && req.End != nil && req.BBox != nil {
		days := req.End.Sub(*req.Start).Hours() / 24
		if days < 1 {
			days = 1
		}
		samples := int(days/5) + 1
		plan.EstimatedPU = EstimatePU(PUInputs{
			BBox:        req.BBox,
			ResolutionM: resolution,
			Bands:       len(plan.Bands),
			Samples:     samples,
		})
		plan.EstimateNote = "assumes a ~5-day revisit; the actual number of cloud-free acquisitions may differ"
	}
	return plan
}

func indexBands(def index.Definition) []string {
	bands := append(append([]string{}, def.Bands...), "SCL")
	return bands
}

// IndexSeries computes a spectral index over an area and time window using the
// Sentinel Hub Statistical API.
func (p *Provider) IndexSeries(ctx context.Context, req provider.IndexRequest) (provider.IndexSeries, error) {
	if !p.SupportsIndex() {
		return provider.IndexSeries{}, provider.ErrNotConfigured
	}

	name := strings.ToLower(strings.TrimSpace(req.Index))
	if name == "" {
		name = "ndvi"
	}
	if req.BBox == nil || req.Start == nil || req.End == nil {
		return provider.IndexSeries{}, fmt.Errorf("index %s requires an area and a time window", name)
	}

	custom := strings.TrimSpace(req.Evalscript) != ""
	var def index.Definition
	if !custom {
		var ok bool
		def, ok = index.Lookup(name)
		if !ok {
			return provider.IndexSeries{}, index.UnknownError(req.Index)
		}
	}

	// A custom evalscript may carry different bands and output id.
	evalscript := def.Evalscript
	outputID := def.OutputID
	if custom {
		evalscript = req.Evalscript
		outputID = req.OutputID
	} else if strings.TrimSpace(req.OutputID) != "" {
		outputID = strings.TrimSpace(req.OutputID)
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

	// Fail early with an actionable message instead of a raw HTTP 400 from
	// Sentinel Hub when the requested output is too large.
	if width, height := OutputDimensions(req.BBox, resolution); width > maxOutputPixels || height > maxOutputPixels {
		return provider.IndexSeries{}, fmt.Errorf(
			"requested area and --resolution %gm produce a %dx%d px output, above the %d px per-side limit: "+
				"increase --resolution (for example %gm) or reduce the area",
			resolution, width, height, maxOutputPixels,
			suggestResolution(req.BBox, resolution, width, height))
	}

	// resx/resy are expressed in the units of the requested CRS. Our bounds
	// use EPSG:4326 (degrees), so convert metres to degrees at the bbox centre.
	resDeg := metresToDegrees(resolution, req.BBox)

	statistics := map[string]any{"default": map[string]any{}}
	if len(req.Percentiles) > 0 {
		statistics["default"] = map[string]any{
			"percentiles": map[string]any{"k": req.Percentiles},
		}
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
			"evalscript":          evalscript,
			"resx":                resDeg,
			"resy":                resDeg,
		},
		"calculations": map[string]any{
			"default": map[string]any{"statistics": statistics},
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

	title := def.Title
	if req.Title != "" {
		title = req.Title
	}
	unit := def.Unit
	if req.Unit != "" {
		unit = req.Unit
	}
	formula := def.Formula
	if req.Formula != "" {
		formula = req.Formula
	}
	if title == "" {
		title = strings.ToUpper(name)
	}

	series := provider.IndexSeries{
		Index:      name,
		Title:      title,
		Unit:       unit,
		Formula:    formula,
		Collection: collection,
		Interval:   interval,
		Intervals:  make([]provider.IndexInterval, 0, len(response.Data)),
	}
	for _, bucket := range response.Data {
		entry := provider.IndexInterval{From: bucket.Interval.From, To: bucket.Interval.To}
		if stats, ok := statisticsFor(bucket.Outputs, outputID); ok {
			entry.Mean = stats.Mean
			entry.Min = stats.Min
			entry.Max = stats.Max
			entry.StDev = stats.StDev
			entry.SampleCount = stats.SampleCount
			entry.Percentiles = stats.Percentiles
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

// maxOutputPixels caps a Statistical API request: Sentinel Hub rejects output
// dimensions above 2500 pixels per side.
const maxOutputPixels = 2500

// OutputDimensions returns the output size in pixels for a request, before any
// API call, so callers can warn or fail early.
func OutputDimensions(bbox *geometry.BBox, resolutionM float64) (int, int) {
	if bbox == nil {
		return 0, 0
	}
	if resolutionM <= 0 {
		resolutionM = 10
	}
	widthM := widthM(bbox.MinLon, bbox.MaxLon, (bbox.MinLat+bbox.MaxLat)/2)
	heightM := haversine((bbox.MaxLat - bbox.MinLat) * 111320)
	return int(widthM / resolutionM), int(heightM / resolutionM)
}

// suggestResolution proposes a resolution that keeps the output within limits.
func suggestResolution(bbox *geometry.BBox, resolutionM float64, width, height int) float64 {
	largest := width
	if height > largest {
		largest = height
	}
	if largest <= maxOutputPixels {
		return resolutionM
	}
	suggested := resolutionM * float64(largest) / float64(maxOutputPixels)
	// Round up to a tidy value.
	return math.Ceil(suggested/5) * 5
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
	Percentiles map[string]float64
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

	if pctRaw, ok := raw["percentiles"]; ok {
		var pct map[string]json.RawMessage
		if err := json.Unmarshal(pctRaw, &pct); err == nil && len(pct) > 0 {
			s.Percentiles = make(map[string]float64, len(pct))
			for key, value := range pct {
				if parsed := flexFloat(value); parsed != nil {
					s.Percentiles[key] = *parsed
				}
			}
			if len(s.Percentiles) == 0 {
				s.Percentiles = nil
			}
		}
	}
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
