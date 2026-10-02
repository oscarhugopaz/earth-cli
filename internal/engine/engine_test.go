package engine

import (
	"strings"
	"testing"

	"github.com/oscarhugopaz/earth-cli/internal/config"
)

func TestNewRegistersCopernicus(t *testing.T) {
	e := New(config.Default())

	names := e.ProviderNames()
	if len(names) != 1 || names[0] != "copernicus" {
		t.Fatalf("ProviderNames = %v", names)
	}

	providers := e.Providers()
	if len(providers) != 1 || providers[0].Name() != "copernicus" {
		t.Fatalf("Providers = %v", providers)
	}
	if providers[0].Type() != "stac" || providers[0].Status() != "available" {
		t.Fatalf("provider metadata = %s/%s", providers[0].Type(), providers[0].Status())
	}
}

func TestUnknownProvider(t *testing.T) {
	e := New(config.Default())
	_, err := e.Provider("nasa")
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownObservation(t *testing.T) {
	e := New(config.Default())
	_, err := e.Resolver("foo")
	if err == nil {
		t.Fatal("err = nil, want error")
	}
	for _, fragment := range []string{"unknown observation", "Available observations", "vegetation"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("err = %q, missing %q", err, fragment)
		}
	}
}

func TestDefaultProvider(t *testing.T) {
	e := New(config.Default())
	p, err := e.DefaultProvider()
	if err != nil {
		t.Fatalf("DefaultProvider returned error: %v", err)
	}
	if p.Name() != "copernicus" {
		t.Fatalf("default provider = %q", p.Name())
	}
}
