package copernicus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

func fastRetryProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	p := newTestProvider(t, handler)
	p.retry = retryConfig{maxAttempts: 3, base: time.Millisecond, max: 3 * time.Millisecond}
	return p
}

func TestSearchRetriesTransientStatus(t *testing.T) {
	var attempts int32
	p := fastRetryProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			http.Error(w, "try later", http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `{"features":[`+featureJSON("a")+`],"links":[]}`)
	})

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len = %d, want 1", len(observations))
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestSearchGivesUpAfterMaxAttempts(t *testing.T) {
	var attempts int32
	p := fastRetryProvider(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, "still unavailable", http.StatusServiceUnavailable)
	})

	_, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 1})
	if err == nil {
		t.Fatal("Search = nil error, want error")
	}
	if !strings.Contains(err.Error(), "HTTP 503 Service Unavailable") {
		t.Fatalf("err = %q", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

func TestSearchDoesNotRetryClientError(t *testing.T) {
	var attempts int32
	p := fastRetryProvider(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	_, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 1})
	if err == nil {
		t.Fatal("Search = nil error, want error")
	}
	if !strings.Contains(err.Error(), "HTTP 400 Bad Request") {
		t.Fatalf("err = %q", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestSearchRetriesNetworkError(t *testing.T) {
	fake := &flakyDoer{failures: 1}
	p := &Provider{
		baseURL: "https://example.test/v1",
		client:  fake,
		retry:   retryConfig{maxAttempts: 3, base: time.Millisecond, max: 2 * time.Millisecond},
	}

	observations, err := p.Search(context.Background(), provider.SearchRequest{Collection: "c", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("len = %d, want 0", len(observations))
	}
	if fake.calls != 2 {
		t.Fatalf("calls = %d, want 2", fake.calls)
	}
}

type flakyDoer struct {
	calls    int
	failures int
}

func (f *flakyDoer) Do(_ *http.Request) (*http.Response, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, errors.New("connection reset by peer")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(`{"features":[],"links":[]}`)),
	}, nil
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty", "", 0},
		{"zero", "0", 0},
		{"negative", "-5", 0},
		{"seconds", "3", 3 * time.Second},
		{"invalid", "soon", 0},
		{"http date", "Fri, 02 Oct 2026 12:00:05 GMT", 5 * time.Second},
		{"past http date", "Fri, 02 Oct 2026 11:59:00 GMT", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseRetryAfter(tc.header, now); got != tc.want {
				t.Fatalf("parseRetryAfter(%q) = %v, want %v", tc.header, got, tc.want)
			}
		})
	}
}

func TestRetryableStatus(t *testing.T) {
	retryable := []int{408, 429, 500, 502, 503, 504}
	for _, code := range retryable {
		if !retryableStatus(code) {
			t.Fatalf("status %d should be retryable", code)
		}
	}
	notRetryable := []int{400, 401, 403, 404, 422}
	for _, code := range notRetryable {
		if retryableStatus(code) {
			t.Fatalf("status %d should not be retryable", code)
		}
	}
}
