package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/config"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func TestConfiguredSTACProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"collections":[{"id":"landsat","title":"Landsat"}]}`))
	}))
	defer server.Close()
	cfg := config.Default()
	cfg.DefaultProvider = "other"
	cfg.Providers["other"] = config.ProviderConfig{STACURL: server.URL, ClientID: "ignored", ClientSecret: "ignored"}
	p, err := New(cfg).DefaultProvider()
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "other" {
		t.Fatal(p.Name())
	}
	if p.(provider.IndexProvider).SupportsIndex() {
		t.Fatal("generic STAC must not process indices")
	}
	collections, err := p.Collections(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(collections) != 1 || collections[0].ID != "landsat" {
		t.Fatalf("collections = %v", collections)
	}
}
