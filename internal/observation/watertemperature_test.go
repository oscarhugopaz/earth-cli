package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestWaterTemperatureObservation(t *testing.T) {
	res, ok := Lookup("water-temperature")
	if !ok {
		t.Fatal("water-temperature observation should be registered")
	}
	if res.Name() != "water-temperature" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("water-temperature", "copernicus")
	if !ok {
		t.Fatal("water-temperature should map to copernicus")
	}
	if !strings.HasPrefix(target.ProcessingCollection, "byoc-") {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if !strings.Contains(target.Evalscript, "LSWT") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.Unit != "°C" {
		t.Fatalf("unit = %q", target.Unit)
	}
	if target.DefaultResolutionM != 1000 {
		t.Fatalf("default resolution = %v", target.DefaultResolutionM)
	}
}

func TestWaterTemperatureResolve(t *testing.T) {
	base := &fakeProvider{name: "copernicus", searchResult: []provider.Observation{{ID: "lswt"}}}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "lswt", Title: "LSWT", Unit: "°C"},
	}
	res, _ := Lookup("water-temperature")

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
	if indexer.last.Resolution != 1000 {
		t.Fatalf("resolution = %v", indexer.last.Resolution)
	}
	if result.Index == nil || result.Index.Unit != "°C" {
		t.Fatalf("result index = %+v", result.Index)
	}
}
