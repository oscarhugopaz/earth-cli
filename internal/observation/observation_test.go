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

type fakeIndexProvider struct {
	*fakeProvider
	series  provider.IndexSeries
	plan    provider.IndexPlan
	err     error
	called  bool
	planned bool
	last    provider.IndexRequest
}

func (f *fakeIndexProvider) SupportsIndex() bool { return true }

func (f *fakeIndexProvider) PlanIndex(req provider.IndexRequest) provider.IndexPlan {
	f.planned = true
	f.last = req
	if f.plan.Index != "" {
		return f.plan
	}
	return provider.IndexPlan{Index: req.Index, Collection: req.Collection, EstimatedPU: 1.23}
}

func (f *fakeIndexProvider) IndexSeries(_ context.Context, req provider.IndexRequest) (provider.IndexSeries, error) {
	f.called = true
	f.last = req
	return f.series, f.err
}

func ptrTime(v time.Time) *time.Time { return &v }
func ptrFloat(v float64) *float64    { return &v }

func TestNamesAndAliases(t *testing.T) {
	names := Names()
	want := []string{
		"aerosol", "atmosphere", "burnt-area", "carbon-monoxide", "chlorophyll", "crop", "flood",
		"land-cover", "methane", "moisture", "ozone", "snow", "soil-moisture", "sulfur-dioxide",
		"temperature", "urban", "vegetation", "water-quality",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("Names = %v, want %v", names, want)
	}

	aliases := map[string]string{
		"fire":  "burnt-area",
		"burn":  "burnt-area",
		"water": "flood",
		"ndvi":  "vegetation",
	}
	for alias, canonical := range aliases {
		res, ok := Lookup(alias)
		if !ok {
			t.Fatalf("alias %q should resolve", alias)
		}
		if res.Name() != canonical {
			t.Fatalf("alias %q -> %q, want %q", alias, res.Name(), canonical)
		}
	}

	if _, ok := Lookup("foo"); ok {
		t.Fatal("unknown observation should not resolve")
	}
}

func TestResolveVegetationDefaultsToNDVI(t *testing.T) {
	fake := &fakeProvider{
		name: "copernicus",
		searchResult: []provider.Observation{
			{ID: "cloudy", DateTime: ptrTime(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)), CloudCover: ptrFloat(40)},
			{ID: "clear", DateTime: ptrTime(time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)), CloudCover: ptrFloat(5)},
		},
	}
	res, _ := Lookup("vegetation")

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	result, err := res.Resolve(context.Background(), fake, Request{BBox: &bbox, PeriodLabel: "last 90 days"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if result.Observation != "vegetation" || result.Collection != "sentinel-2-l2a" {
		t.Fatalf("result = %+v", result)
	}
	if result.Scenes != 2 || result.CloudScenes != 2 {
		t.Fatalf("scenes = %d/%d", result.Scenes, result.CloudScenes)
	}
	if result.BestScene == nil || result.BestScene.ID != "clear" {
		t.Fatalf("best scene = %+v", result.BestScene)
	}
	if result.Index != nil {
		t.Fatal("no credentials -> no index")
	}
	if result.Note == "" {
		t.Fatal("note should explain that no index was computed")
	}
}

func TestResolveFloodUsesMNDWI(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "mndwi", Title: "MNDWI", Formula: "(B03 - B11) / (B03 + B11)"},
	}
	res, _ := Lookup("flood")

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Index != "mndwi" {
		t.Fatalf("index = %q, want mndwi", indexer.last.Index)
	}
	if result.Index == nil || result.Index.Index != "mndwi" {
		t.Fatalf("result index = %+v", result.Index)
	}
	if strings.Join(result.Bands, ",") != "B03,B11" {
		t.Fatalf("bands = %v", result.Bands)
	}
}

func TestResolveBurntAreaUsesNBR(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "nbr", Title: "NBR"},
	}
	res, _ := Lookup("fire") // alias

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{BBox: &bbox, Start: start, End: end})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Index != "nbr" {
		t.Fatalf("index = %q, want nbr", indexer.last.Index)
	}
	if result.Observation != "burnt-area" {
		t.Fatalf("observation = %q", result.Observation)
	}
}

func TestResolveIndexOverride(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{
		fakeProvider: base,
		series:       provider.IndexSeries{Index: "ndwi", Title: "NDWI"},
	}
	res, _ := Lookup("flood")

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	if _, err := res.Resolve(context.Background(), indexer, Request{
		BBox: &bbox, Start: start, End: end, Index: "ndwi",
	}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if indexer.last.Index != "ndwi" {
		t.Fatalf("index = %q, want the override ndwi", indexer.last.Index)
	}
}

func TestResolveRejectsUnknownIndexOverride(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{fakeProvider: base}
	res, _ := Lookup("vegetation")

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	_, err := res.Resolve(context.Background(), indexer, Request{
		BBox: &bbox, Start: start, End: end, Index: "not-real",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown index") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveDryRunPlansWithoutComputing(t *testing.T) {
	base := &fakeProvider{name: "copernicus"}
	indexer := &fakeIndexProvider{fakeProvider: base}
	res, _ := Lookup("vegetation")

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := ptrTime(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	end := ptrTime(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))

	result, err := res.Resolve(context.Background(), indexer, Request{
		BBox: &bbox, Start: start, End: end, DryRun: true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !indexer.planned || indexer.called {
		t.Fatalf("planned=%v called=%v", indexer.planned, indexer.called)
	}
	if result.IndexPlan == nil || result.Index != nil {
		t.Fatalf("plan = %+v index = %+v", result.IndexPlan, result.Index)
	}
}

func TestResolveUnknownProvider(t *testing.T) {
	res, _ := Lookup("vegetation")
	_, err := res.Resolve(context.Background(), &fakeProvider{name: "nasa"}, Request{})
	if err == nil || !strings.Contains(err.Error(), "not available from provider") {
		t.Fatalf("err = %v", err)
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
