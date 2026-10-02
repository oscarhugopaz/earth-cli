package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestSoilMoistureObservation(t *testing.T) {
	res, ok := Lookup("soil-moisture")
	if !ok {
		t.Fatal("soil-moisture observation should be registered")
	}
	if res.Name() != "soil-moisture" {
		t.Fatalf("name = %q", res.Name())
	}

	target, ok := TargetFor("soil-moisture", "copernicus")
	if !ok {
		t.Fatal("soil-moisture should map to copernicus")
	}
	if !strings.HasPrefix(target.ProcessingCollection, "byoc-") {
		t.Fatalf("processing collection should be a BYOC id, got %q", target.ProcessingCollection)
	}
	if strings.HasSuffix(target.Collection, "_cog") == false {
		t.Fatalf("STAC collection should be the COG id, got %q", target.Collection)
	}
	if !strings.Contains(target.Evalscript, "SSM") {
		t.Fatalf("evalscript = %q", target.Evalscript)
	}
	if target.Unit != "% saturation" {
		t.Fatalf("unit = %q", target.Unit)
	}
}

func TestSoilMoistureResolveUsesBYOC(t *testing.T) {
	base := &fakeProvider{
		name:         "copernicus",
		searchResult: []provider.Observation{{ID: "ssm"}},
	}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series: provider.IndexSeries{
			Index: "ssm", Title: "SSM", Unit: "% saturation",
		},
	}
	res, _ := Lookup("soil-moisture")

	bbox := geometry.BBox{MinLon: -4, MinLat: 40, MaxLon: 3, MaxLat: 44}
	start := ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(indexer.last.Collection, "byoc-") {
		t.Fatalf("index collection = %q", indexer.last.Collection)
	}
	if indexer.last.Resolution != 1000 {
		t.Fatalf("default resolution = %v", indexer.last.Resolution)
	}
	if result.Index == nil || result.Index.Unit != "% saturation" {
		t.Fatalf("result index = %+v", result.Index)
	}
	if !strings.Contains(result.Note, "CLMS") {
		t.Fatalf("note = %q", result.Note)
	}
}
