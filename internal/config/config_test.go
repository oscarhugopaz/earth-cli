package config

import (
	"os"
	"path/filepath"
	"testing"
)

func lookupFrom(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.DefaultProvider != DefaultProviderName {
		t.Fatalf("DefaultProvider = %q", cfg.DefaultProvider)
	}
	if got := cfg.STACURL(DefaultProviderName); got != DefaultCopernicusSTACURL {
		t.Fatalf("STACURL = %q", got)
	}
}

func TestLoadWithoutConfigFile(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{"HOME": t.TempDir()}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DefaultProvider != DefaultProviderName {
		t.Fatalf("DefaultProvider = %q", cfg.DefaultProvider)
	}
}

func TestLoadConfigFile(t *testing.T) {
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(xdg, "earth", "config.yaml"), `
default_provider: copernicus
providers:
  copernicus:
    stac_url: https://example.test/stac
`)

	cfg, err := Load(lookupFrom(map[string]string{"XDG_CONFIG_HOME": xdg}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := cfg.STACURL("copernicus"); got != "https://example.test/stac" {
		t.Fatalf("STACURL = %q", got)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(xdg, "earth", "config.yaml"), `
default_provider: other
providers:
  copernicus:
    stac_url: https://example.test/stac
`)

	cfg, err := Load(lookupFrom(map[string]string{
		"XDG_CONFIG_HOME":           xdg,
		"EARTH_PROVIDER":            "copernicus",
		"EARTH_COPERNICUS_STAC_URL": "https://env.test/stac",
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DefaultProvider != "copernicus" {
		t.Fatalf("DefaultProvider = %q", cfg.DefaultProvider)
	}
	if got := cfg.STACURL("copernicus"); got != "https://env.test/stac" {
		t.Fatalf("STACURL = %q", got)
	}
}

func TestLoadMalformedConfigFile(t *testing.T) {
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(xdg, "earth", "config.yaml"), "default_provider: [unterminated")

	if _, err := Load(lookupFrom(map[string]string{"XDG_CONFIG_HOME": xdg})); err == nil {
		t.Fatal("Load = nil error, want error")
	}
}

func TestLoadCopernicusCredentialsFromEnv(t *testing.T) {
	cfg, err := Load(lookupFrom(map[string]string{
		"EARTH_COPERNICUS_CLIENT_ID":      "my-id",
		"EARTH_COPERNICUS_CLIENT_SECRET":  "my-secret",
		"EARTH_COPERNICUS_TOKEN_URL":      "https://token.test",
		"EARTH_COPERNICUS_STATISTICS_URL": "https://stats.test",
	}))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	provider := cfg.Provider("copernicus")
	if provider.ClientID != "my-id" || provider.ClientSecret != "my-secret" {
		t.Fatalf("credentials = %q/%q", provider.ClientID, provider.ClientSecret)
	}
	if provider.TokenURL != "https://token.test" || provider.StatisticsURL != "https://stats.test" {
		t.Fatalf("urls = %q/%q", provider.TokenURL, provider.StatisticsURL)
	}
}

func TestPath(t *testing.T) {
	if got := Path(lookupFrom(map[string]string{"XDG_CONFIG_HOME": "/xdg"})); got != "/xdg/earth/config.yaml" {
		t.Fatalf("Path = %q", got)
	}
	if got := Path(lookupFrom(map[string]string{"HOME": "/home/me"})); got != "/home/me/.config/earth/config.yaml" {
		t.Fatalf("Path = %q", got)
	}
	if got := Path(lookupFrom(map[string]string{})); got != "" {
		t.Fatalf("Path = %q, want empty", got)
	}
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
