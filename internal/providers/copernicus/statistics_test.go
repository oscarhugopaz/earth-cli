package copernicus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/geometry"
	"github.com/oscarhugopaz/earth-cli/internal/index"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func newStatisticsProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p := New(
		server.URL,
		WithHTTPClient(server.Client()),
		WithCredentials("client-id", "client-secret"),
		WithTokenURL(server.URL+"/token"),
		WithStatisticsURL(server.URL+"/statistics"),
	)
	p.retry = retryConfig{maxAttempts: 2, base: time.Millisecond, max: 2 * time.Millisecond}
	return p
}

func TestIndexSeriesParsingAndAuth(t *testing.T) {
	var tokenCalls int32
	var captured map[string]any
	var mu sync.Mutex
	p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			atomic.AddInt32(&tokenCalls, 1)
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if r.Form.Get("grant_type") != "client_credentials" ||
				r.Form.Get("client_id") != "client-id" ||
				r.Form.Get("client_secret") != "client-secret" {
				http.Error(w, "bad credentials", http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"access_token":"test-token","expires_in":3600,"token_type":"Bearer"}`)

		case "/statistics":
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				http.Error(w, "missing bearer", http.StatusUnauthorized)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			captured = body
			mu.Unlock()
			fmt.Fprint(w, `{
				"status": "OK",
				"data": [
					{"interval": {"from": "2026-09-01T00:00:00Z", "to": "2026-09-11T00:00:00Z"},
					 "outputs": {"ndvi": {"bands": {"B0": {"stats": {"mean": 0.5, "min": 0.1, "max": 0.9, "stDev": 0.12, "sampleCount": 1234}}}}}},
					{"interval": {"from": "2026-09-11T00:00:00Z", "to": "2026-09-21T00:00:00Z"},
					 "outputs": {}}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	})

	if !p.SupportsIndex() {
		t.Fatal("SupportsIndex should be true with credentials")
	}

	bbox := geometry.BBox{MinLon: -70.700, MinLat: -33.600, MaxLon: -70.690, MaxLat: -33.592}
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)

	series, err := p.IndexSeries(context.Background(), provider.IndexRequest{
		Collection: "sentinel-2-l2a",
		Index:      "ndvi",
		BBox:       &bbox,
		Start:      &start,
		End:        &end,
	})
	if err != nil {
		t.Fatalf("IndexSeries returned error: %v", err)
	}
	if len(series.Intervals) != 2 {
		t.Fatalf("intervals = %+v", series.Intervals)
	}
	first := series.Intervals[0]
	if first.Mean == nil || *first.Mean != 0.5 || first.SampleCount == nil || *first.SampleCount != 1234 {
		t.Fatalf("first interval = %+v", first)
	}
	if series.Intervals[1].Mean != nil {
		t.Fatalf("second interval should have no stats: %+v", series.Intervals[1])
	}

	// The evalscript and bounds must be part of the request.
	mu.Lock()
	requestBody := captured
	mu.Unlock()
	aggregation, _ := requestBody["aggregation"].(map[string]any)
	script, _ := aggregation["evalscript"].(string)
	if !strings.Contains(script, "B08") || !strings.Contains(script, "B04") {
		t.Fatalf("evalscript = %q", script)
	}
	input, _ := requestBody["input"].(map[string]any)
	bounds, _ := input["bounds"].(map[string]any)
	if bounds["bbox"] == nil {
		t.Fatalf("request missing bbox: %+v", requestBody)
	}

	// A second call must reuse the cached token.
	if _, err := p.IndexSeries(context.Background(), provider.IndexRequest{Collection: "sentinel-2-l2a", Index: "ndvi", BBox: &bbox, Start: &start, End: &end}); err != nil {
		t.Fatalf("second IndexSeries returned error: %v", err)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token endpoint calls = %d, want 1", got)
	}
}

func TestIndexSeriesWithoutCredentials(t *testing.T) {
	p := New("https://example.test/v1")
	if p.SupportsIndex() {
		t.Fatal("SupportsIndex should be false without credentials")
	}
	_, err := p.IndexSeries(context.Background(), provider.IndexRequest{Index: "ndvi"})
	if !errors.Is(err, provider.ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestIndexSeriesRejectsUnsupportedIndex(t *testing.T) {
	p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("the API must not be called for an unknown index without a custom evalscript")
	})
	bbox := geometry.BBox{MinLon: -70.700, MinLat: -33.600, MaxLon: -70.690, MaxLat: -33.592}
	start, end := fixedWindow()
	_, err := p.IndexSeries(context.Background(), provider.IndexRequest{
		Index: "not-an-index", BBox: &bbox, Start: &start, End: &end, Resolution: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown index") {
		t.Fatalf("err = %v", err)
	}
}

func TestIndexSeriesSupportedIndexes(t *testing.T) {
	// Every registered index must be accepted (and reach the window check),
	// proving the evalscript catalog is wired up.
	for _, name := range index.Names() {
		p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"status":"OK","data":[]}`)
		})
		_, err := p.IndexSeries(context.Background(), provider.IndexRequest{Index: name})
		if err == nil || !strings.Contains(err.Error(), "requires an area and a time window") {
			t.Fatalf("index %q: err = %v", name, err)
		}
	}
}

func TestIndexSeriesRequiresWindow(t *testing.T) {
	p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"OK","data":[]}`)
	})
	_, err := p.IndexSeries(context.Background(), provider.IndexRequest{Index: "ndvi"})
	if err == nil || !strings.Contains(err.Error(), "requires an area and a time window") {
		t.Fatalf("err = %v", err)
	}
}

func TestAuthenticationError(t *testing.T) {
	p := newStatisticsProvider(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_client", http.StatusUnauthorized)
	})
	bbox := geometry.BBox{MinLon: 0, MinLat: 0, MaxLon: 0.01, MaxLat: 0.01}
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	_, err := p.IndexSeries(context.Background(), provider.IndexRequest{Index: "ndvi", BBox: &bbox, Start: &start, End: &end})
	if err == nil || !strings.Contains(err.Error(), "copernicus authentication failed") {
		t.Fatalf("err = %v", err)
	}
}
