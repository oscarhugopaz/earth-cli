package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestAtmosphereObservation(t *testing.T) {
	res, ok := Lookup("atmosphere")
	if !ok {
		t.Fatal("atmosphere observation should be registered")
	}
	if res.Name() != "atmosphere" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("atmosphere", "copernicus")
	if !ok {
		t.Fatal("atmosphere should map to copernicus")
	}
	if target.Collection != AtmosphereCollection {
		t.Fatalf("STAC collection = %q", target.Collection)
	}
	if target.ProcessingCollection != "sentinel-5p-l2" {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if !strings.Contains(target.Evalscript, "NO2") || !strings.Contains(target.Evalscript, "dataMask") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.Unit == "" {
		t.Fatal("atmosphere should declare a unit")
	}
	if target.DefaultResolutionM <= 0 {
		t.Fatal("atmosphere should declare a default resolution")
	}
}

func TestAtmosphereResolve(t *testing.T) {
	base := &fakeProvider{
		name:         "copernicus",
		searchResult: []provider.Observation{{ID: "s5p", CloudCover: ptrFloat(5)}},
	}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series: provider.IndexSeries{
			Index:   "no2",
			Title:   "NO2",
			Unit:    "mol/m²",
			Formula: "Sentinel-5P L2 NO2 vertical column",
		},
	}
	res, _ := Lookup("atmosphere")

	bbox := geometry.BBox{MinLon: -70.75, MinLat: -33.55, MaxLon: -70.55, MaxLat: -33.40}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if base.lastRequest.Collection != AtmosphereCollection {
		t.Fatalf("search collection = %q", base.lastRequest.Collection)
	}
	if indexer.last.Collection != "sentinel-5p-l2" {
		t.Fatalf("index collection = %q", indexer.last.Collection)
	}
	if indexer.last.Resolution != 3500 {
		t.Fatalf("default resolution = %v, want 3500", indexer.last.Resolution)
	}
	if result.Index == nil || result.Index.Unit != "mol/m²" {
		t.Fatalf("result index = %+v", result.Index)
	}
	// The note must name the real source, not Sentinel-3.
	if !strings.Contains(result.Note, "Sentinel-5P") {
		t.Fatalf("note = %q", result.Note)
	}
}

func TestTemperatureDefaultResolution(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{fakeProvider: base, series: provider.IndexSeries{Index: "lst"}}
	res, _ := Lookup("temperature")

	bbox := geometry.BBox{MinLon: -72, MinLat: -36, MaxLon: -69, MaxLat: -32}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC))

	if _, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Resolution != 1000 {
		t.Fatalf("default resolution = %v, want 1000", indexer.last.Resolution)
	}
}

func TestTemperatureNoteNamesSource(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "lst", Title: "LST"},
	}
	res, _ := Lookup("temperature")

	bbox := geometry.BBox{MinLon: -72, MinLat: -36, MaxLon: -69, MaxLat: -32}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(result.Note, "Sentinel-3 SLSTR") {
		t.Fatalf("note = %q", result.Note)
	}
}
