package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestLandCoverObservation(t *testing.T) {
	if _, ok := Lookup("land-cover"); !ok {
		t.Fatal("land-cover observation should be registered")
	}

	target, ok := TargetFor("land-cover", "copernicus")
	if !ok {
		t.Fatal("land-cover should map to copernicus")
	}
	if !strings.HasPrefix(target.ProcessingCollection, "byoc-") {
		t.Fatalf("processing collection = %q", target.ProcessingCollection)
	}
	if target.DefaultInterval != "P1Y" {
		t.Fatalf("default interval = %q, want P1Y", target.DefaultInterval)
	}
	if len(target.ClassLabels) == 0 {
		t.Fatal("land-cover should define class labels")
	}
	if target.ClassLabels[40] != "Cropland" || target.ClassLabels[50] != "Urban / built-up" {
		t.Fatalf("class labels look wrong: %v", target.ClassLabels)
	}
}

func TestLandCoverRequestsPercentiles(t *testing.T) {
	base := &fakeProvider{name: "copernicus", searchResult: []provider.Observation{{ID: "lc"}}}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "land-cover"},
	}
	res, _ := Lookup("land-cover")

	bbox := geometry.BBox{MinLon: 5, MinLat: 45, MaxLon: 5.2, MaxLat: 45.2}
	start := ptrTime(time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// Categorical product: percentiles must be requested to get the dominant class.
	if len(indexer.last.Percentiles) == 0 {
		t.Fatal("land-cover should request percentiles")
	}
	if indexer.last.Interval != "" {
		// The resolver leaves the interval empty and lets the provider default
		// apply; the command supplies it when the user overrides.
		t.Logf("interval passed to provider: %q", indexer.last.Interval)
	}
	if result.ClassLabels == nil {
		t.Fatal("result should carry class labels for formatting")
	}
}

func TestNamesIncludeLandCover(t *testing.T) {
	if _, ok := Lookup("land-cover"); !ok {
		t.Fatal("land-cover missing")
	}
}
