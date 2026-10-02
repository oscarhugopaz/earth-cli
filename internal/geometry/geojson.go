package geometry

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// LoadArea reads a GeoJSON file and derives a bounding box from it.
func LoadArea(path string) (BBox, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BBox{}, fmt.Errorf("read area file %q: %w", path, err)
	}
	bbox, err := BBoxFromGeoJSON(data)
	if err != nil {
		return BBox{}, fmt.Errorf("parse area file %q: %w", path, err)
	}
	return bbox, nil
}

// BBoxFromGeoJSON derives a bounding box from a GeoJSON Polygon, MultiPolygon,
// Feature, or FeatureCollection. True intersects-based searching can replace
// this later without touching the rest of the CLI.
func BBoxFromGeoJSON(data []byte) (BBox, error) {
	var root struct {
		Type        string            `json:"type"`
		Coordinates json.RawMessage   `json:"coordinates"`
		Geometry    json.RawMessage   `json:"geometry"`
		Features    []json.RawMessage `json:"features"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return BBox{}, fmt.Errorf("invalid GeoJSON: %w", err)
	}

	switch root.Type {
	case "Polygon", "MultiPolygon":
		return bboxFromGeometry(data)
	case "Feature":
		return bboxFromGeometry(root.Geometry)
	case "FeatureCollection":
		if len(root.Features) == 0 {
			return BBox{}, fmt.Errorf("FeatureCollection contains no features")
		}
		var union BBox
		haveUnion := false
		for i, feature := range root.Features {
			bbox, err := bboxFromFeature(feature)
			if err != nil {
				return BBox{}, fmt.Errorf("feature %d: %w", i, err)
			}
			if !haveUnion {
				union, haveUnion = bbox, true
				continue
			}
			union = unionOf(union, bbox)
		}
		return union, nil
	case "":
		return BBox{}, fmt.Errorf("invalid GeoJSON: missing \"type\"")
	default:
		return BBox{}, fmt.Errorf("unsupported GeoJSON type %q: expected Polygon, MultiPolygon, Feature, or FeatureCollection", root.Type)
	}
}

func bboxFromFeature(raw json.RawMessage) (BBox, error) {
	var feature struct {
		Type     string          `json:"type"`
		Geometry json.RawMessage `json:"geometry"`
	}
	if err := json.Unmarshal(raw, &feature); err != nil {
		return BBox{}, fmt.Errorf("invalid feature: %w", err)
	}
	if feature.Type != "" && feature.Type != "Feature" {
		return BBox{}, fmt.Errorf("unsupported type %q: expected Feature", feature.Type)
	}
	return bboxFromGeometry(feature.Geometry)
}

func bboxFromGeometry(raw json.RawMessage) (BBox, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return BBox{}, fmt.Errorf("feature has no geometry")
	}
	var geom struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &geom); err != nil {
		return BBox{}, fmt.Errorf("invalid geometry: %w", err)
	}
	switch geom.Type {
	case "Polygon", "MultiPolygon":
	default:
		return BBox{}, fmt.Errorf("unsupported geometry type %q: expected Polygon or MultiPolygon", geom.Type)
	}

	minLon, minLat := math.Inf(1), math.Inf(1)
	maxLon, maxLat := math.Inf(-1), math.Inf(-1)
	found := false

	var walk func(json.RawMessage) error
	walk = func(node json.RawMessage) error {
		var point []float64
		if err := json.Unmarshal(node, &point); err == nil && len(point) >= 2 {
			found = true
			minLon = math.Min(minLon, point[0])
			minLat = math.Min(minLat, point[1])
			maxLon = math.Max(maxLon, point[0])
			maxLat = math.Max(maxLat, point[1])
			return nil
		}

		var children []json.RawMessage
		if err := json.Unmarshal(node, &children); err != nil {
			return fmt.Errorf("invalid coordinate array: %w", err)
		}
		for _, child := range children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}

	if err := walk(geom.Coordinates); err != nil {
		return BBox{}, err
	}
	if !found {
		return BBox{}, fmt.Errorf("geometry contains no coordinates")
	}

	bbox := BBox{MinLon: minLon, MinLat: minLat, MaxLon: maxLon, MaxLat: maxLat}
	if err := bbox.Validate(); err != nil {
		return BBox{}, fmt.Errorf("derived bounding box is invalid: %w", err)
	}
	return bbox, nil
}

func unionOf(a, b BBox) BBox {
	return BBox{
		MinLon: math.Min(a.MinLon, b.MinLon),
		MinLat: math.Min(a.MinLat, b.MinLat),
		MaxLon: math.Max(a.MaxLon, b.MaxLon),
		MaxLat: math.Max(a.MaxLat, b.MaxLat),
	}
}
