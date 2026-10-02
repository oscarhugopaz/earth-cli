// Package geometry provides small helpers for bounding boxes and GeoJSON
// areas. It deliberately avoids native geospatial dependencies so the CLI
// remains a lightweight standalone binary.
package geometry

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// BBox is a geographic bounding box in WGS84 degrees.
type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// ParseBBox parses a "minLon,minLat,maxLon,maxLat" string.
func ParseBBox(raw string) (BBox, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return BBox{}, fmt.Errorf("invalid bbox %q: expected minLon,minLat,maxLon,maxLat", raw)
	}

	values := make([]float64, 4)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		value, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return BBox{}, fmt.Errorf("invalid bbox %q: %q is not a number", raw, part)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return BBox{}, fmt.Errorf("invalid bbox %q: %q is not a finite number", raw, part)
		}
		values[i] = value
	}

	bbox := BBox{MinLon: values[0], MinLat: values[1], MaxLon: values[2], MaxLat: values[3]}
	if err := bbox.Validate(); err != nil {
		return BBox{}, fmt.Errorf("invalid bbox %q: %w", raw, err)
	}
	return bbox, nil
}

// Validate checks coordinate ranges and ordering.
func (b BBox) Validate() error {
	if b.MinLon < -180 || b.MaxLon > 180 {
		return fmt.Errorf("longitude must be within -180..180")
	}
	if b.MinLat < -90 || b.MaxLat > 90 {
		return fmt.Errorf("latitude must be within -90..90")
	}
	if b.MinLon >= b.MaxLon {
		return fmt.Errorf("minLon must be less than maxLon")
	}
	if b.MinLat >= b.MaxLat {
		return fmt.Errorf("minLat must be less than maxLat")
	}
	return nil
}

// Array returns the bbox as a STAC-compatible [minLon, minLat, maxLon, maxLat].
func (b BBox) Array() [4]float64 {
	return [4]float64{b.MinLon, b.MinLat, b.MaxLon, b.MaxLat}
}

// Slice returns the bbox as a []float64 suitable for JSON encoding.
func (b BBox) Slice() []float64 {
	arr := b.Array()
	return arr[:]
}

// String renders the bbox in minLon,minLat,maxLon,maxLat form.
func (b BBox) String() string {
	return fmt.Sprintf("%g,%g,%g,%g", b.MinLon, b.MinLat, b.MaxLon, b.MaxLat)
}
