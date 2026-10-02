package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestAerosolObservation(t *testing.T) {
	res, ok := Lookup("aerosol")
	if !ok {
		t.Fatal("aerosol observation should be registered")
	}
	if res.Name() != "aerosol" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("aerosol", "copernicus")
	if !ok {
		t.Fatal("aerosol should map to copernicus")
	}
	if target.Collection != "sentinel-5p-l2-aer-ai" {
		t.Fatalf("STAC collection = %q", target.Collection)
	}
	if target.ProcessingCollection != "sentinel-5p-l2" {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if !strings.Contains(target.Evalscript, "AER_AI_340_380") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.DefaultResolutionM != 3500 {
		t.Fatalf("default resolution = %v", target.DefaultResolutionM)
	}
}

func TestAerosolResolve(t *testing.T) {
	base := &fakeProvider{name: "copernicus", searchResult: []provider.Observation{{ID: "ai"}}}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "aerosol-index", Title: "Aerosol Index"},
	}
	res, _ := Lookup("aerosol")

	bbox := geometry.BBox{MinLon: -60, MinLat: -10, MaxLon: -50, MaxLat: 0}
	start := ptrTime(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Collection != "sentinel-5p-l2" {
		t.Fatalf("index collection = %q", indexer.last.Collection)
	}
	if result.Index == nil || result.Index.Index != "aerosol-index" {
		t.Fatalf("result index = %+v", result.Index)
	}
}
