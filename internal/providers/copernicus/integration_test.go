package copernicus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// These tests hit the live Copernicus Data Space Ecosystem STAC API. They are
// opt-in so normal unit tests never depend on the network:
//
//	EARTH_INTEGRATION=1 go test ./internal/providers/copernicus/
//
// Do not hammer the service: keep the queries small.
func integrationProvider(t *testing.T) *Provider {
	t.Helper()
	if os.Getenv("EARTH_INTEGRATION") != "1" {
		t.Skip("set EARTH_INTEGRATION=1 to run live Copernicus STAC integration tests")
	}
	return New(DefaultSTACURL, WithTimeout(30*time.Second))
}

func integrationContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIntegrationCollections(t *testing.T) {
	p := integrationProvider(t)
	collections, err := p.Collections(integrationContext(t), 0)
	if err != nil {
		t.Fatalf("Collections returned error: %v", err)
	}
	if len(collections) == 0 {
		t.Fatal("expected at least one collection")
	}
	found := false
	for _, collection := range collections {
		if collection.ID == "sentinel-2-l2a" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("sentinel-2-l2a not found among %d collections", len(collections))
	}
}

func TestIntegrationCollection(t *testing.T) {
	p := integrationProvider(t)
	collection, err := p.Collection(integrationContext(t), "sentinel-2-l2a")
	if err != nil {
		t.Fatalf("Collection returned error: %v", err)
	}
	if collection.ID != "sentinel-2-l2a" || collection.Provider != "copernicus" {
		t.Fatalf("collection = %+v", collection)
	}
	if collection.Title == "" {
		t.Fatal("expected a title")
	}
}

func TestIntegrationCollectionNotFound(t *testing.T) {
	p := integrationProvider(t)
	_, err := p.Collection(integrationContext(t), "this-collection-does-not-exist")
	if err == nil {
		t.Fatal("expected an error for a missing collection")
	}
}

func TestIntegrationSearchPaginates(t *testing.T) {
	p := integrationProvider(t)
	bbox := geometry.BBox{MinLon: -75, MinLat: -40, MaxLon: -65, MaxLat: -30}
	start := time.Now().UTC().AddDate(0, -6, 0)
	end := time.Now().UTC()

	const limit = 120
	observations, err := p.Search(integrationContext(t), provider.SearchRequest{
		Collection: "sentinel-2-l2a",
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
		Limit:      limit,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != limit {
		t.Fatalf("len = %d, want %d (pagination)", len(observations), limit)
	}
	seen := make(map[string]struct{}, len(observations))
	for _, observation := range observations {
		if observation.ID == "" {
			t.Fatal("observation without id")
		}
		if _, duplicate := seen[observation.ID]; duplicate {
			t.Fatalf("duplicate observation %q across pages", observation.ID)
		}
		seen[observation.ID] = struct{}{}
	}
}

func TestIntegrationItem(t *testing.T) {
	p := integrationProvider(t)
	ctx := integrationContext(t)

	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}
	start := time.Now().UTC().AddDate(0, -3, 0)
	end := time.Now().UTC()

	observations, err := p.Search(ctx, provider.SearchRequest{
		Collection: "sentinel-2-l2a",
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
		Limit:      1,
	})
	if err != nil || len(observations) == 0 {
		t.Fatalf("search for an item failed: err=%v n=%d", err, len(observations))
	}

	item, err := p.Item(ctx, "sentinel-2-l2a", observations[0].ID)
	if err != nil {
		t.Fatalf("Item returned error: %v", err)
	}
	if item.ID != observations[0].ID {
		t.Fatalf("item id = %q, want %q", item.ID, observations[0].ID)
	}
	if len(item.AssetDetails) == 0 {
		t.Fatal("expected asset details")
	}
}
