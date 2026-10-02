// Package config resolves configuration from defaults, an optional XDG config
// file, and environment variables. earth works with no configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultProviderName is used when nothing else is configured.
	DefaultProviderName = "copernicus"
	// DefaultCopernicusSTACURL is the public CDSE STAC endpoint.
	DefaultCopernicusSTACURL = "https://stac.dataspace.copernicus.eu/v1"

	envProvider      = "EARTH_PROVIDER"
	envCopernicusURL = "EARTH_COPERNICUS_STAC_URL"
)

// Provider holds per-provider configuration.
type Provider struct {
	STACURL string `yaml:"stac_url"`
}

// Config is the effective configuration.
type Config struct {
	DefaultProvider string
	Providers       map[string]Provider
}

// fileSchema mirrors the YAML config file.
type fileSchema struct {
	DefaultProvider string              `yaml:"default_provider"`
	Providers       map[string]Provider `yaml:"providers"`
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		DefaultProvider: DefaultProviderName,
		Providers: map[string]Provider{
			DefaultProviderName: {STACURL: DefaultCopernicusSTACURL},
		},
	}
}

// STACURL returns the configured STAC endpoint for a provider.
func (c Config) STACURL(providerName string) string {
	if p, ok := c.Providers[providerName]; ok && strings.TrimSpace(p.STACURL) != "" {
		return strings.TrimSpace(p.STACURL)
	}
	if providerName == DefaultProviderName {
		return DefaultCopernicusSTACURL
	}
	return ""
}

// Load resolves configuration. lookup is os.LookupEnv in production and a stub
// in tests. A missing config file is not an error.
func Load(lookup func(string) (string, bool)) (Config, error) {
	cfg := Default()

	path := Path(lookup)
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			var parsed fileSchema
			if err := yaml.Unmarshal(data, &parsed); err != nil {
				return Config{}, fmt.Errorf("parse config %s: %w", path, err)
			}
			merge(&cfg, parsed)
		case errors.Is(err, os.ErrNotExist):
			// No config file: defaults are fine.
		default:
			return Config{}, fmt.Errorf("read config %s: %w", path, err)
		}
	}

	applyEnv(&cfg, lookup)
	return cfg, nil
}

// Path returns the config file location following XDG conventions.
func Path(lookup func(string) (string, bool)) string {
	if xdg, ok := lookup("XDG_CONFIG_HOME"); ok && strings.TrimSpace(xdg) != "" {
		return filepath.Join(strings.TrimSpace(xdg), "earth", "config.yaml")
	}
	if home, ok := lookup("HOME"); ok && strings.TrimSpace(home) != "" {
		return filepath.Join(strings.TrimSpace(home), ".config", "earth", "config.yaml")
	}
	return ""
}

func merge(cfg *Config, parsed fileSchema) {
	if strings.TrimSpace(parsed.DefaultProvider) != "" {
		cfg.DefaultProvider = strings.TrimSpace(parsed.DefaultProvider)
	}
	for name, provider := range parsed.Providers {
		if strings.TrimSpace(provider.STACURL) == "" {
			continue
		}
		if cfg.Providers == nil {
			cfg.Providers = map[string]Provider{}
		}
		cfg.Providers[name] = provider
	}
}

func applyEnv(cfg *Config, lookup func(string) (string, bool)) {
	if value, ok := lookup(envProvider); ok && strings.TrimSpace(value) != "" {
		cfg.DefaultProvider = strings.TrimSpace(value)
	}
	if value, ok := lookup(envCopernicusURL); ok && strings.TrimSpace(value) != "" {
		if cfg.Providers == nil {
			cfg.Providers = map[string]Provider{}
		}
		cfg.Providers[DefaultProviderName] = Provider{STACURL: strings.TrimSpace(value)}
	}
}
