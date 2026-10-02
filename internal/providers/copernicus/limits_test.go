package copernicus

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func fixedWindow() (time.Time, time.Time) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	return start, end
}

func TestOutputDimensions(t *testing.T) {
	// ~1.1 km at 10 m is about 110 px per side.
	bbox := &geometry.BBox{MinLon: -70.70, MinLat: -33.60, MaxLon: -70.69, MaxLat: -33.592}
	width, height := OutputDimensions(bbox, 10)
	if width < 90 || width > 130 {
		t.Fatalf("width = %d, want ~110", width)
	}
	if height < 70 || height > 110 {
		t.Fatalf("height = %d", height)
	}

	// Coarser resolution halves the pixel count roughly.
	coarseW, _ := OutputDimensions(bbox, 20)
	if coarseW >= width {
		t.Fatalf("coarser resolution should shrink output: %d vs %d", coarseW, width)
	}

	if w, h := OutputDimensions(nil, 10); w != 0 || h != 0 {
		t.Fatalf("nil bbox = %dx%d", w, h)
	}
}

func TestIndexSeriesRejectsOversizedRequest(t *testing.T) {
	p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("the API must not be called for an oversized request")
	})
	// A one-degree box at 10 m is far above the 2500 px per-side limit.
	bbox := geometry.BBox{MinLon: 0, MinLat: 0, MaxLon: 1, MaxLat: 1}
	start, end := fixedWindow()
	_, err := p.IndexSeries(context.Background(), provider.IndexRequest{
		Index: "ndvi", BBox: &bbox, Start: &start, End: &end, Resolution: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "per-side limit") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "increase --resolution") {
		t.Fatalf("error should suggest a fix: %v", err)
	}
}
