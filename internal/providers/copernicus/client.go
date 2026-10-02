// Package copernicus implements the Copernicus Data Space Ecosystem (CDSE)
// provider using its public STAC API for discovery and the Sentinel Hub
// Statistical API for derived indices.
package copernicus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

const (
	// DefaultSTACURL is the public, anonymous CDSE STAC endpoint.
	DefaultSTACURL = "https://stac.dataspace.copernicus.eu/v1"
	// DefaultTokenURL is the CDSE OAuth token endpoint (client credentials).
	DefaultTokenURL = "https://identity.dataspace.copernicus.eu/auth/realms/CDSE/protocol/openid-connect/token"
	// DefaultStatisticsURL is the Sentinel Hub Statistical API endpoint.
	// CDSE migrated the path structure to /statistics/v1 on the sh. host;
	// the old statistics.dataspace.copernicus.eu host no longer resolves.
	DefaultStatisticsURL = "https://sh.dataspace.copernicus.eu/statistics/v1"

	name        = "copernicus"
	displayName = "Copernicus"

	maxResponseBytes = 32 << 20 // 32 MiB safety cap
	maxErrorBody     = 200
)

type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Provider implements provider.Provider (and provider.IndexProvider when
// credentials are configured) for Copernicus.
type Provider struct {
	baseURL       string
	statisticsURL string
	client        doer
	retry         retryConfig

	tokenURL     string
	clientID     string
	clientSecret string

	auth *tokenSource
}

// Option customizes a Provider.
type Option func(*Provider)

// WithHTTPClient replaces the HTTP client (useful in tests).
func WithHTTPClient(client doer) Option {
	return func(p *Provider) {
		if client != nil {
			p.client = client
		}
	}
}

// WithTimeout sets the HTTP client timeout when the default client is used.
func WithTimeout(d time.Duration) Option {
	return func(p *Provider) {
		if d <= 0 {
			return
		}
		if hc, ok := p.client.(*http.Client); ok {
			hc.Timeout = d
		}
	}
}

// WithCredentials enables authenticated processing (Standard/Statistical API).
func WithCredentials(clientID, clientSecret string) Option {
	return func(p *Provider) {
		p.clientID = strings.TrimSpace(clientID)
		p.clientSecret = strings.TrimSpace(clientSecret)
	}
}

// WithTokenURL overrides the OAuth token endpoint.
func WithTokenURL(rawURL string) Option {
	return func(p *Provider) {
		p.tokenURL = strings.TrimSpace(rawURL)
	}
}

// WithStatisticsURL overrides the Statistical API endpoint.
func WithStatisticsURL(rawURL string) Option {
	return func(p *Provider) {
		p.statisticsURL = strings.TrimSpace(rawURL)
	}
}

// New builds a Copernicus provider. An empty baseURL falls back to the public
// CDSE STAC endpoint.
func New(baseURL string, opts ...Option) *Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultSTACURL
	}
	p := &Provider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		retry:   defaultRetryConfig(),
	}
	for _, opt := range opts {
		opt(p)
	}

	if p.tokenURL == "" {
		p.tokenURL = DefaultTokenURL
	}
	if p.statisticsURL == "" {
		p.statisticsURL = DefaultStatisticsURL
	}
	if p.clientID != "" && p.clientSecret != "" {
		p.auth = &tokenSource{
			clientID:     p.clientID,
			clientSecret: p.clientSecret,
			tokenURL:     p.tokenURL,
			client:       p.client,
		}
	}
	return p
}

// BaseURL returns the configured STAC endpoint.
func (p *Provider) BaseURL() string { return p.baseURL }

func (p *Provider) Name() string   { return name }
func (p *Provider) Type() string   { return "stac" }
func (p *Provider) Status() string { return "available" }

func (p *Provider) Description() string {
	return "Copernicus Data Space Ecosystem public STAC API"
}

// do performs an unauthenticated HTTP request, retrying transient failures.
func (p *Provider) do(ctx context.Context, method, rawURL string, body []byte, out any) error {
	return p.request(ctx, method, rawURL, body, "", out)
}

// request performs an HTTP request (optionally bearer-authenticated), retrying
// transient failures with backoff.
func (p *Provider) request(ctx context.Context, method, rawURL string, body []byte, bearer string, out any) error {
	attempts := p.retry.maxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader(body))
		if err != nil {
			return fmt.Errorf("build %s STAC request: %w", displayName, err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "earth-cli")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s STAC request failed: %w", displayName, err)
			if ctx.Err() != nil || attempt == attempts {
				return lastErr
			}
			if !sleepOrDone(ctx, p.backoff(attempt, "")) {
				return ctx.Err()
			}
			continue
		}

		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()

		responseErr := &provider.ResponseError{
			Provider: displayName,
			Method:   method,
			URL:      rawURL,
			Status:   resp.Status,
			Code:     resp.StatusCode,
			Body:     snippet(data),
		}

		if readErr != nil {
			lastErr = fmt.Errorf("read %s STAC response: %w", displayName, readErr)
			if ctx.Err() != nil || attempt == attempts {
				return lastErr
			}
			if !sleepOrDone(ctx, p.backoff(attempt, "")) {
				return ctx.Err()
			}
			continue
		}

		if retryableStatus(resp.StatusCode) {
			lastErr = responseErr
			if attempt == attempts {
				return lastErr
			}
			if !sleepOrDone(ctx, p.backoff(attempt, resp.Header.Get("Retry-After"))) {
				return ctx.Err()
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return responseErr
		}

		if out == nil {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s STAC response: %w", displayName, err)
		}
		return nil
	}

	return lastErr
}

func bodyReader(body []byte) io.Reader {
	if body == nil {
		return nil
	}
	return bytes.NewReader(body)
}

func snippet(data []byte) string {
	text := strings.Join(strings.Fields(string(data)), " ")
	if len(text) > maxErrorBody {
		text = text[:maxErrorBody] + "…"
	}
	return text
}

func encodePathSegment(segment string) string {
	return url.PathEscape(segment)
}
