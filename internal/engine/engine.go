// Package engine wires providers and observation resolvers together. It is the
// seam that keeps the CLI independent from any single Earth observation
// catalog.
package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/config"
	"github.com/oscarhugopaz/earth-cli/internal/observation"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
	"github.com/oscarhugopaz/earth-cli/internal/providers/copernicus"
)

// Engine holds the registered providers and resolvers.
type Engine struct {
	providers       map[string]provider.Provider
	resolvers       map[string]observation.Resolver
	defaultProvider string
}

type options struct {
	httpTimeout time.Duration
}

// Option customizes an Engine.
type Option func(*options)

// WithHTTPTimeout sets the per-request HTTP timeout for providers.
func WithHTTPTimeout(d time.Duration) Option {
	return func(o *options) { o.httpTimeout = d }
}

// New builds an Engine from configuration.
func New(cfg config.Config, opts ...Option) *Engine {
	resolved := options{httpTimeout: 30 * time.Second}
	for _, opt := range opts {
		opt(&resolved)
	}

	e := &Engine{
		providers:       map[string]provider.Provider{},
		resolvers:       map[string]observation.Resolver{},
		defaultProvider: cfg.DefaultProvider,
	}
	if e.defaultProvider == "" {
		e.defaultProvider = config.DefaultProviderName
	}

	e.RegisterProvider(copernicus.New(cfg.STACURL("copernicus"), copernicus.WithTimeout(resolved.httpTimeout)))
	for name, resolver := range observation.Resolvers() {
		e.RegisterResolver(name, resolver)
	}
	return e
}

// RegisterProvider adds or replaces a provider.
func (e *Engine) RegisterProvider(p provider.Provider) {
	e.providers[p.Name()] = p
}

// RegisterResolver adds or replaces an observation resolver.
func (e *Engine) RegisterResolver(name string, resolver observation.Resolver) {
	e.resolvers[name] = resolver
}

// Providers returns registered providers sorted by name.
func (e *Engine) Providers() []provider.Provider {
	out := make([]provider.Provider, 0, len(e.providers))
	for _, p := range e.providers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// ProviderNames returns registered provider names sorted by name.
func (e *Engine) ProviderNames() []string {
	names := make([]string, 0, len(e.providers))
	for name := range e.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Provider looks up a provider by name.
func (e *Engine) Provider(name string) (provider.Provider, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	p, ok := e.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q; available providers: %s", name, strings.Join(e.ProviderNames(), ", "))
	}
	return p, nil
}

// DefaultProvider returns the configured default provider.
func (e *Engine) DefaultProvider() (provider.Provider, error) {
	return e.Provider(e.defaultProvider)
}

// Resolvers returns registered resolvers sorted by name.
func (e *Engine) Resolvers() []observation.Resolver {
	out := make([]observation.Resolver, 0, len(e.resolvers))
	for _, resolver := range e.resolvers {
		out = append(out, resolver)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Resolver looks up an observation resolver by name.
func (e *Engine) Resolver(name string) (observation.Resolver, error) {
	resolver, ok := e.resolvers[strings.TrimSpace(name)]
	if !ok {
		return nil, fmt.Errorf("unknown observation %q\n\nAvailable observations:\n  %s",
			name, strings.Join(observation.Names(), "\n  "))
	}
	return resolver, nil
}

// Observe resolves a semantic observation, selecting the provider when none is
// given.
func (e *Engine) Observe(ctx context.Context, observationName, providerName string, req observation.Request) (*observation.Result, error) {
	resolver, err := e.Resolver(observationName)
	if err != nil {
		return nil, err
	}

	var selected provider.Provider
	if strings.TrimSpace(providerName) == "" {
		selected, err = e.DefaultProvider()
	} else {
		selected, err = e.Provider(providerName)
	}
	if err != nil {
		return nil, err
	}

	return resolver.Resolve(ctx, selected, req)
}
