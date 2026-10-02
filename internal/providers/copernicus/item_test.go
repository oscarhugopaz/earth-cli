package copernicus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestItemNormalizationAndAssets(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/sentinel-2-l2a/items/S2_1" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{
			"id": "S2_1",
			"collection": "sentinel-2-l2a",
			"bbox": [-71,-34,-70,-33],
			"geometry": {"type": "Polygon", "coordinates": [[[0,0],[1,0],[1,1],[0,1],[0,0]]]},
			"properties": {"datetime": "2026-09-14T14:37:39.024Z", "eo:cloud_cover": 3.2},
			"assets": {
				"thumbnail": {"href": "https://example.test/t.png", "type": "image/png"},
				"B04_10m": {"href": "https://example.test/B04.tif", "type": "image/tiff; application=geotiff", "title": "Band 4", "roles": ["data"]}
			},
			"links": [{"rel": "self", "href": "https://example.test/items/S2_1"}]
		}`)
	})

	item, err := p.Item(context.Background(), "sentinel-2-l2a", "S2_1")
	if err != nil {
		t.Fatalf("Item returned error: %v", err)
	}
	if item.ID != "S2_1" || item.Collection != "sentinel-2-l2a" {
		t.Fatalf("item = %+v", item)
	}
	if item.CloudCover == nil || *item.CloudCover != 3.2 {
		t.Fatalf("CloudCover = %v", item.CloudCover)
	}
	if item.ItemURL != "https://example.test/items/S2_1" {
		t.Fatalf("ItemURL = %q", item.ItemURL)
	}
	if strings.Join(item.Assets, ",") != "B04_10m,thumbnail" {
		t.Fatalf("Assets = %v", item.Assets)
	}
	if len(item.AssetDetails) != 2 {
		t.Fatalf("AssetDetails = %+v", item.AssetDetails)
	}
	first := item.AssetDetails[0]
	if first.Name != "B04_10m" || first.Href != "https://example.test/B04.tif" || strings.Join(first.Roles, ",") != "data" {
		t.Fatalf("first asset = %+v", first)
	}
}

func TestItemNotFound(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	_, err := p.Item(context.Background(), "sentinel-2-l2a", "missing")
	if err == nil || !strings.Contains(err.Error(), `"missing" was not found`) {
		t.Fatalf("err = %v", err)
	}
}

func TestItemRequiresArguments(t *testing.T) {
	p := New("")
	if _, err := p.Item(context.Background(), "", "id"); err == nil {
		t.Fatal("expected error for empty collection")
	}
	if _, err := p.Item(context.Background(), "c", ""); err == nil {
		t.Fatal("expected error for empty id")
	}
}
