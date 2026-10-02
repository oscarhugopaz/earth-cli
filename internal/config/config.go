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

	envProvider             = "EARTH_PROVIDER"
	envCopernicusURL        = "EARTH_COPERNICUS_STAC_URL"
	envCopernicusClientID   = "EARTH_COPERNICUS_CLIENT_ID"
	envCopernicusSecret     = "EARTH_COPERNICUS_CLIENT_SECRET"
	envCopernicusTokenURL   = "EARTH_COPERNICUS_TOKEN_URL"
	envCopernicusStatistics = "EARTH_COPERNICUS_STATISTICS_URL"
)

// ProviderConfig holds configuration for a single provider.
//
// ClientID/ClientSecret enable authenticated processing APIs and must never be
// committed. Prefer the environment over the config file for secrets.
type ProviderConfig struct {
	STACURL       string `yaml:"stac_url"`
	ClientID      string `yaml:"client_id,omitempty"`
	ClientSecret  string `yaml:"client_secret,omitempty"`
	TokenURL      string `yaml:"token_url,omitempty"`
	StatisticsURL string `yaml:"statistics_url,omitempty"`
}

// Config is the effective configuration.
type Config struct {
	DefaultProvider string
	Providers       map[string]ProviderConfig
}

// fileSchema mirrors the YAML config file.
type fileSchema struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		DefaultProvider: DefaultProviderName,
		Providers: map[string]ProviderConfig{
			DefaultProviderName: {STACURL: DefaultCopernicusSTACURL},
		},
	}
}

// Provider returns the configuration for a provider, applying defaults.
func (c Config) Provider(name string) ProviderConfig {
	if p, ok := c.Providers[name]; ok {
		return p
	}
	if name == DefaultProviderName {
		return ProviderConfig{STACURL: DefaultCopernicusSTACURL}
	}
	return ProviderConfig{}
}

// STACURL returns the configured STAC endpoint for a provider.
func (c Config) STACURL(providerName string) string {
	url := strings.TrimSpace(c.Provider(providerName).STACURL)
	if url != "" {
		return url
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
		if cfg.Providers == nil {
			cfg.Providers = map[string]ProviderConfig{}
		}
		cfg.Providers[name] = provider
	}
}

func applyEnv(cfg *Config, lookup func(string) (string, bool)) {
	if value, ok := lookup(envProvider); ok && strings.TrimSpace(value) != "" {
		cfg.DefaultProvider = strings.TrimSpace(value)
	}

	update := func(key string, set func(*ProviderConfig, string)) {
		value, ok := lookup(key)
		if !ok || strings.TrimSpace(value) == "" {
			return
		}
		if cfg.Providers == nil {
			cfg.Providers = map[string]ProviderConfig{}
		}
		provider := cfg.Providers[DefaultProviderName]
		set(&provider, strings.TrimSpace(value))
		cfg.Providers[DefaultProviderName] = provider
	}

	update(envCopernicusURL, func(p *ProviderConfig, v string) { p.STACURL = v })
	update(envCopernicusClientID, func(p *ProviderConfig, v string) { p.ClientID = v })
	update(envCopernicusSecret, func(p *ProviderConfig, v string) { p.ClientSecret = v })
	update(envCopernicusTokenURL, func(p *ProviderConfig, v string) { p.TokenURL = v })
	update(envCopernicusStatistics, func(p *ProviderConfig, v string) { p.StatisticsURL = v })
}
