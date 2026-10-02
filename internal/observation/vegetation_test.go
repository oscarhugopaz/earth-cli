package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

type fakeProvider struct {
	name         string
	searchResult []provider.Observation
	searchErr    error
	lastRequest  provider.SearchRequest
}

func (f *fakeProvider) Name() string        { return f.name }
func (f *fakeProvider) Type() string        { return "fake" }
func (f *fakeProvider) Status() string      { return "available" }
func (f *fakeProvider) Description() string { return "fake provider" }
func (f *fakeProvider) Collections(context.Context, int) ([]provider.Collection, error) {
	return nil, nil
}
func (f *fakeProvider) Collection(context.Context, string) (provider.Collection, error) {
	return provider.Collection{}, nil
}
func (f *fakeProvider) Item(context.Context, string, string) (provider.Observation, error) {
	return provider.Observation{}, nil
}
func (f *fakeProvider) Search(_ context.Context, req provider.SearchRequest) ([]provider.Observation, error) {
	f.lastRequest = req
	return f.searchResult, f.searchErr
}

func ptrTime(value time.Time) *time.Time { return &value }
func ptrFloat(value float64) *float64    { return &value }

func TestVegetationResolve(t *testing.T) {
	fake := &fakeProvider{
		name: "copernicus",
		searchResult: []provider.Observation{
			{ID: "cloudy", DateTime: ptrTime(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)), CloudCover: ptrFloat(40)},
			{ID: "clear", DateTime: ptrTime(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)), CloudCover: ptrFloat(5), ItemURL: "https://example.test/clear"},
			{ID: "mid", DateTime: ptrTime(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)), CloudCover: ptrFloat(10)},
		},
	}

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	result, err := (Vegetation{}).Resolve(context.Background(), fake, Request{
		BBox:        &bbox,
		PeriodLabel: "last 90 days",
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if result.Observation != "vegetation" || result.Provider != "copernicus" {
		t.Fatalf("result = %+v", result)
	}
	if result.Collection != "sentinel-2-l2a" || result.Source != "Sentinel-2 Level-2A" {
		t.Fatalf("mapping = %+v", result)
	}
	if result.Scenes != 3 || result.CloudScenes != 3 {
		t.Fatalf("scene counts = %d/%d", result.Scenes, result.CloudScenes)
	}
	if result.MeanCloudCover == nil || *result.MeanCloudCover < 18.33 || *result.MeanCloudCover > 18.34 {
		t.Fatalf("MeanCloudCover = %v", result.MeanCloudCover)
	}
	if result.BestScene == nil || result.BestScene.ID != "clear" {
		t.Fatalf("BestScene = %+v", result.BestScene)
	}
	if len(result.Bands) != 0 {
		t.Fatalf("Bands should be empty without an index, got %v", result.Bands)
	}
	if result.Formula != "" {
		t.Fatalf("Formula should be empty without an index, got %q", result.Formula)
	}
	if result.Note == "" {
		t.Fatal("Note should explain that no index was computed")
	}
	if len(result.Items) != 3 {
		t.Fatalf("Items = %d", len(result.Items))
	}
	if fake.lastRequest.Collection != "sentinel-2-l2a" {
		t.Fatalf("search collection = %q", fake.lastRequest.Collection)
	}
	if fake.lastRequest.Limit != 100 {
		t.Fatalf("default search limit = %d, want 100", fake.lastRequest.Limit)
	}
}

func TestVegetationResolveUnknownProvider(t *testing.T) {
	fake := &fakeProvider{name: "nasa"}
	_, err := (Vegetation{}).Resolve(context.Background(), fake, Request{})
	if err == nil || !strings.Contains(err.Error(), "not available from provider") {
		t.Fatalf("err = %v", err)
	}
}

// fakeIndexProvider adds provider.IndexProvider on top of fakeProvider.
type fakeIndexProvider struct {
	*fakeProvider
	series  provider.IndexSeries
	err     error
	called  bool
	planned bool
	last    provider.IndexRequest
}

func (f *fakeIndexProvider) SupportsIndex() bool { return true }

func (f *fakeIndexProvider) PlanIndex(req provider.IndexRequest) provider.IndexPlan {
	f.planned = true
	return provider.IndexPlan{Index: req.Index, Collection: req.Collection, EstimatedPU: 1.23}
}

func (f *fakeIndexProvider) IndexSeries(_ context.Context, req provider.IndexRequest) (provider.IndexSeries, error) {
	f.called = true
	f.last = req
	return f.series, f.err
}

func TestVegetationResolveComputesNDVI(t *testing.T) {
	base := &fakeProvider{
		name:         "copernicus",
		searchResult: []provider.Observation{{ID: "a", CloudCover: ptrFloat(5)}},
	}
	mean := 0.52
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series: provider.IndexSeries{
			Index:     "ndvi",
			Title:     "NDVI",
			Formula:   "(B08 - B04) / (B08 + B04)",
			Interval:  "P10D",
			Intervals: []provider.IndexInterval{{Mean: &mean}},
		},
	}

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := (Vegetation{}).Resolve(context.Background(), indexer, Request{
		BBox:  &bbox,
		Start: &start,
		End:   &end,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result.Index == nil || len(result.Index.Intervals) != 1 {
		t.Fatalf("NDVI = %+v", result.Index)
	}
	if !indexer.called || indexer.last.Index != "ndvi" {
		t.Fatalf("index request = %+v (called=%v)", indexer.last, indexer.called)
	}
	if result.Note != indexNote("NDVI") {
		t.Fatalf("Note = %q", result.Note)
	}
}

func TestVegetationResolveWithoutCredentialsKeepsNote(t *testing.T) {
	// fakeProvider does not implement IndexProvider: metadata only.
	fake := &fakeProvider{name: "copernicus"}
	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := (Vegetation{}).Resolve(context.Background(), fake, Request{BBox: &bbox, Start: &start, End: &end})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result.Index != nil {
		t.Fatalf("NDVI should be nil, got %+v", result.Index)
	}
	if result.Note != noIndexNote {
		t.Fatalf("Note = %q", result.Note)
	}
}

func TestVegetationResolveDryRunPlansWithoutComputing(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{fakeProvider: base}

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	result, err := (Vegetation{}).Resolve(context.Background(), indexer, Request{
		BBox:   &bbox,
		Start:  &start,
		End:    &end,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !indexer.planned {
		t.Fatal("PlanIndex should have been called")
	}
	if indexer.called {
		t.Fatal("IndexSeries must not be called in dry-run mode")
	}
	if result.Index != nil {
		t.Fatalf("NDVI should be nil in dry run, got %+v", result.Index)
	}
	if result.IndexPlan == nil || result.IndexPlan.EstimatedPU == 0 {
		t.Fatalf("NDVIPlan = %+v", result.IndexPlan)
	}
	if result.Note != dryRunNote {
		t.Fatalf("Note = %q", result.Note)
	}
}

func TestVegetationResolvePassesIntervalAndResolution(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "ndvi"},
	}

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if _, err := (Vegetation{}).Resolve(context.Background(), indexer, Request{
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
		Interval:   "P30D",
		Resolution: 20,
	}); err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if indexer.last.Interval != "P30D" || indexer.last.Resolution != 20 {
		t.Fatalf("index request = %+v", indexer.last)
	}
}

func TestSelectBestSceneFallsBackToNewest(t *testing.T) {
	older := provider.Observation{ID: "older", DateTime: ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))}
	newer := provider.Observation{ID: "newer", DateTime: ptrTime(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC))}

	best := selectBestScene([]provider.Observation{older, newer})
	if best == nil || best.ID != "newer" {
		t.Fatalf("best = %+v", best)
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 1 || names[0] != "vegetation" {
		t.Fatalf("Names = %v", names)
	}
	if _, ok := Lookup("vegetation"); !ok {
		t.Fatal("Lookup(vegetation) should succeed")
	}
	if _, ok := Lookup("flood"); ok {
		t.Fatal("Lookup(flood) should fail")
	}
}
