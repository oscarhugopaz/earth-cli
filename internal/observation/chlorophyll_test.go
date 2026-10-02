package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestChlorophyllObservation(t *testing.T) {
	res, ok := Lookup("chlorophyll")
	if !ok {
		t.Fatal("chlorophyll observation should be registered")
	}
	if res.Name() != "chlorophyll" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("chlorophyll", "copernicus")
	if !ok {
		t.Fatal("chlorophyll should map to copernicus")
	}
	if target.ProcessingCollection != "sentinel-3-olci-l2" {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if target.Collection != "sentinel-3-olci-2-wfr-ntc" {
		t.Fatalf("STAC collection = %q", target.Collection)
	}
	if !strings.Contains(target.Evalscript, "CHL_OC4ME") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.Unit != "mg/m³" {
		t.Fatalf("unit = %q", target.Unit)
	}
}

func TestChlorophyllResolve(t *testing.T) {
	base := &fakeProvider{name: "copernicus", searchResult: []provider.Observation{{ID: "olci"}}}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "chl", Title: "CHL", Unit: "mg/m³"},
	}
	res, _ := Lookup("chlorophyll")

	bbox := geometry.BBox{MinLon: 10.4, MinLat: 45.5, MaxLon: 10.8, MaxLat: 45.8}
	start := ptrTime(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Collection != "sentinel-3-olci-l2" {
		t.Fatalf("index collection = %q", indexer.last.Collection)
	}
	if indexer.last.Resolution != 300 {
		t.Fatalf("resolution = %v", indexer.last.Resolution)
	}
	if result.Index == nil || result.Index.Unit != "mg/m³" {
		t.Fatalf("result index = %+v", result.Index)
	}
}
