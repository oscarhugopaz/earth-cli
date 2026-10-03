package copernicus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// tokenSource obtains and caches an OAuth2 client-credentials access token.
// The token is kept in memory only: it is never written to disk.
type tokenSource struct {
	clientID     string
	clientSecret string
	tokenURL     string
	client       doer

	mu     sync.Mutex
	token  string
	expiry time.Time
}

func (t *tokenSource) accessToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.token != "" && time.Until(t.expiry) > time.Minute {
		return t.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", t.clientID)
	form.Set("client_secret", t.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build Copernicus authentication request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "earth-cli")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("copernicus authentication failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read Copernicus authentication response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("copernicus authentication failed: HTTP %s: %s", resp.Status, snippet(data))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("decode Copernicus authentication response: %w", err)
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("copernicus authentication returned no access token")
	}

	lifetime := time.Duration(payload.ExpiresIn) * time.Second
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	t.token = payload.AccessToken
	t.expiry = time.Now().Add(lifetime)
	return t.token, nil
}
