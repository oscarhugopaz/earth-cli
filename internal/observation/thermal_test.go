package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestTemperatureObservation(t *testing.T) {
	res, ok := Lookup("temperature")
	if !ok {
		t.Fatal("temperature observation should be registered")
	}
	if res.Name() != "temperature" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("temperature", "copernicus")
	if !ok {
		t.Fatal("temperature should map to copernicus")
	}
	if target.Collection != LSTCollection {
		t.Fatalf("STAC collection = %q, want %q", target.Collection, LSTCollection)
	}
	if target.ProcessingCollection != "sentinel-3-slstr-l2" {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if !target.Thermal {
		t.Fatal("temperature should be marked thermal")
	}
	// The evalscript must read LST and convert to Celsius, without SCL.
	if !strings.Contains(target.Evalscript, "LST") || !strings.Contains(target.Evalscript, "273.15") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if strings.Contains(target.Evalscript, "SCL") {
		t.Fatal("thermal evalscript should not use a Sentinel-2 SCL band")
	}
}

func TestTemperatureResolveUsesProcessingCollection(t *testing.T) {
	base := &fakeProvider{
		name:         "copernicus",
		searchResult: []provider.Observation{{ID: "s3", CloudCover: ptrFloat(10)}},
	}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series: provider.IndexSeries{
			Index:   "lst",
			Title:   "LST",
			Unit:    "°C",
			Formula: "LST (K) - 273.15 → °C",
		},
	}
	res, _ := Lookup("temperature")

	bbox := geometry.BBox{MinLon: -72, MinLat: -36, MaxLon: -69, MaxLat: -32}
	start := ptrTime(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// The search uses the STAC collection; the index call uses the processing
	// collection id, which differs.
	if base.lastRequest.Collection != LSTCollection {
		t.Fatalf("search collection = %q, want %q", base.lastRequest.Collection, LSTCollection)
	}
	if indexer.last.Collection != "sentinel-3-slstr-l2" {
		t.Fatalf("index collection = %q, want sentinel-3-slstr-l2", indexer.last.Collection)
	}
	if indexer.last.Evalscript == "" || indexer.last.OutputID != "lst" {
		t.Fatalf("index request = %+v", indexer.last)
	}
	if result.Index == nil || result.Index.Unit != "°C" {
		t.Fatalf("result index = %+v", result.Index)
	}
	if result.Bands != nil {
		t.Fatalf("thermal observation should not report spectral bands: %v", result.Bands)
	}
}

func TestThermalDryRun(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{fakeProvider: base}
	res, _ := Lookup("temperature")

	bbox := geometry.BBox{MinLon: -72, MinLat: -36, MaxLon: -69, MaxLat: -32}
	start := ptrTime(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{
		BBox: &bbox, Start: start, End: end, DryRun: true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !indexer.planned || indexer.called {
		t.Fatalf("planned=%v called=%v", indexer.planned, indexer.called)
	}
	if result.IndexPlan == nil {
		t.Fatal("dry run should produce a plan")
	}
}

func TestNamesIncludesTemperature(t *testing.T) {
	names := strings.Join(Names(), ",")
	if !strings.Contains(names, "temperature") {
		t.Fatalf("Names = %q", names)
	}
}
