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
	if strings.Join(result.Bands, ",") != "B04,B08" {
		t.Fatalf("Bands = %v", result.Bands)
	}
	if result.Formula != "NDVI = (B08 - B04) / (B08 + B04)" {
		t.Fatalf("Formula = %q", result.Formula)
	}
	if result.Note == "" {
		t.Fatal("Note should explain that NDVI is not computed")
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
