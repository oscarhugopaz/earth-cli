package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestWaterQualityObservation(t *testing.T) {
	res, ok := Lookup("water-quality")
	if !ok {
		t.Fatal("water-quality observation should be registered")
	}
	if res.Name() != "water-quality" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("water-quality", "copernicus")
	if !ok {
		t.Fatal("water-quality should map to copernicus")
	}
	if !strings.HasPrefix(target.ProcessingCollection, "byoc-") {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if !strings.Contains(target.Evalscript, "TSI") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.DefaultResolutionM != 300 {
		t.Fatalf("default resolution = %v", target.DefaultResolutionM)
	}
}

func TestWaterQualityResolve(t *testing.T) {
	base := &fakeProvider{name: "copernicus", searchResult: []provider.Observation{{ID: "lwq"}}}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "tsi", Title: "TSI", Unit: "TSI"},
	}
	res, _ := Lookup("water-quality")

	bbox := geometry.BBox{MinLon: 10.4, MinLat: 45.5, MaxLon: 10.8, MaxLat: 45.8}
	start := ptrTime(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(indexer.last.Collection, "byoc-") {
		t.Fatalf("index collection = %q", indexer.last.Collection)
	}
	if indexer.last.Resolution != 300 {
		t.Fatalf("resolution = %v", indexer.last.Resolution)
	}
	if result.Index == nil || result.Index.Unit != "TSI" {
		t.Fatalf("result index = %+v", result.Index)
	}
}
