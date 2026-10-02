package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p := New(server.URL, WithHTTPClient(server.Client()))
	// Keep retry behavior but with near-zero delays so tests stay fast.
	p.retry = retryConfig{maxAttempts: 3, base: time.Millisecond, max: 2 * time.Millisecond}
	return p
}

func TestCollectionsPaginationAndNormalization(t *testing.T) {
	var providerRef *Provider
	providerRef = newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("offset") == "" {
			fmt.Fprintf(w, `{
				"collections": [{
					"id": "sentinel-2-l2a",
					"title": "Sentinel-2 Level-2A",
					"description": "desc",
					"license": "other",
					"keywords": ["Sentinel"],
					"extent": {"spatial": {"bbox": [[-180,-90,180,90]]}, "temporal": {"interval": [["2015-06-27T10:25:31Z", null]]}},
					"links": [
						{"rel": "items", "href": "%s/collections/sentinel-2-l2a/items"},
						{"rel": "http://www.opengis.net/def/rel/ogc/1.0/queryables", "href": "%s/collections/sentinel-2-l2a/queryables"}
					]
				}],
				"links": [{"rel": "next", "href": "%s/collections?offset=1"}]
			}`, providerRef.BaseURL(), providerRef.BaseURL(), providerRef.BaseURL())
			return
		}
		fmt.Fprintf(w, `{
			"collections": [{"id": "sentinel-1-grd", "title": "Sentinel-1 GRD"}],
			"links": []
		}`)
	})

	collections, err := providerRef.Collections(context.Background(), 0)
	if err != nil {
		t.Fatalf("Collections returned error: %v", err)
	}
	if len(collections) != 2 {
		t.Fatalf("len(collections) = %d, want 2", len(collections))
	}

	first := collections[0]
	if first.ID != "sentinel-2-l2a" || first.Provider != "copernicus" {
		t.Fatalf("first collection = %+v", first)
	}
	if first.ItemURL != providerRef.BaseURL()+"/collections/sentinel-2-l2a/items" {
		t.Fatalf("ItemURL = %q", first.ItemURL)
	}
	if !strings.HasSuffix(first.QueryablesURL, "/queryables") {
		t.Fatalf("QueryablesURL = %q", first.QueryablesURL)
	}
	if first.Extent == nil || len(first.Extent.Spatial) != 1 || len(first.Extent.Temporal) != 1 {
		t.Fatalf("Extent = %+v", first.Extent)
	}

	if collections[1].ID != "sentinel-1-grd" {
		t.Fatalf("second collection id = %q", collections[1].ID)
	}
}

func TestCollectionsHonorsLimit(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"collections":[{"id":"a"},{"id":"b"},{"id":"c"}],"links":[]}`)
	})

	collections, err := p.Collections(context.Background(), 2)
	if err != nil {
		t.Fatalf("Collections returned error: %v", err)
	}
	if len(collections) != 2 {
		t.Fatalf("len = %d, want 2", len(collections))
	}
}

func TestCollectionNotFound(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, err := p.Collection(context.Background(), "missing")
	if err == nil {
		t.Fatal("Collection = nil error, want error")
	}
	if !strings.Contains(err.Error(), `"missing" was not found`) {
		t.Fatalf("error = %q", err)
	}
}

func TestSearchNormalization(t *testing.T) {
	var captured map[string]any
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/search" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{
			"type": "FeatureCollection",
			"features": [{
				"id": "S2_1",
				"collection": "sentinel-2-l2a",
				"bbox": [-71,-34,-70,-33],
				"geometry": {"type": "Polygon", "coordinates": [[[0,0],[1,0],[1,1],[0,1],[0,0]]]},
				"properties": {"datetime": "2026-09-14T14:37:39.024Z", "eo:cloud_cover": 0.14},
				"assets": {"B08_10m": {}, "B04_10m": {}},
				"links": [{"rel": "self", "href": "https://example.test/items/S2_1"}]
			}]
		}`)
	})

	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 23, 59, 59, 0, time.UTC)
	bbox := geometry.BBox{MinLon: -70.8, MinLat: -33.6, MaxLon: -70.4, MaxLat: -33.3}

	observations, err := p.Search(context.Background(), provider.SearchRequest{
		Collection: "sentinel-2-l2a",
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
		Limit:      7,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len = %d, want 1", len(observations))
	}

	got := observations[0]
	if got.ID != "S2_1" || got.Provider != "copernicus" {
		t.Fatalf("observation = %+v", got)
	}
	if got.CloudCover == nil || *got.CloudCover != 0.14 {
		t.Fatalf("CloudCover = %v", got.CloudCover)
	}
	if got.DateTime == nil || !got.DateTime.Equal(time.Date(2026, 9, 14, 14, 37, 39, 24000000, time.UTC)) {
		t.Fatalf("DateTime = %v", got.DateTime)
	}
	if strings.Join(got.Assets, ",") != "B04_10m,B08_10m" {
		t.Fatalf("Assets = %v", got.Assets)
	}
	if got.ItemURL != "https://example.test/items/S2_1" {
		t.Fatalf("ItemURL = %q", got.ItemURL)
	}

	if captured["datetime"] != "2026-09-01T00:00:00Z/2026-10-01T23:59:59Z" {
		t.Fatalf("datetime request = %v", captured["datetime"])
	}
	if captured["limit"] != float64(7) {
		t.Fatalf("limit request = %v", captured["limit"])
	}
	if _, ok := captured["collections"]; !ok {
		t.Fatalf("missing collections in request: %v", captured)
	}
}

func TestSearchAcceptsStringCloudCover(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"features":[{"id":"a","properties":{"datetime":"2026-09-14T00:00:00Z","eo:cloud_cover":"12.5"}}]}`)
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if observations[0].CloudCover == nil || *observations[0].CloudCover != 12.5 {
		t.Fatalf("CloudCover = %v", observations[0].CloudCover)
	}
}

func TestResponseError(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	})

	_, err := p.Collections(context.Background(), 1)
	if err == nil {
		t.Fatal("Collections = nil error, want error")
	}
	if !strings.Contains(err.Error(), "Copernicus STAC request failed: HTTP 503 Service Unavailable") {
		t.Fatalf("error = %q", err)
	}
}

func TestNewDefaultsBaseURL(t *testing.T) {
	p := New("")
	if p.BaseURL() != DefaultSTACURL {
		t.Fatalf("BaseURL = %q", p.BaseURL())
	}
}
