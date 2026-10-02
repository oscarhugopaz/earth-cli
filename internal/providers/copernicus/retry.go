package copernicus

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxRetryAfter caps how long a server can ask us to wait, so a single retry
// cannot stall the CLI indefinitely.
const maxRetryAfter = 30 * time.Second

type retryConfig struct {
	maxAttempts int
	base        time.Duration
	max         time.Duration
}

func defaultRetryConfig() retryConfig {
	return retryConfig{
		maxAttempts: 4,
		base:        500 * time.Millisecond,
		max:         8 * time.Second,
	}
}

// retryableStatus reports whether an HTTP status is worth retrying. Client
// errors other than 408/429 are not retried: they will not fix themselves.
func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// backoff computes the delay before the next attempt, honoring Retry-After and
// adding jitter to avoid synchronized retries.
func (p *Provider) backoff(attempt int, retryAfterHeader string) time.Duration {
	base := p.retry.base
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	maxDelay := p.retry.max
	if maxDelay <= 0 {
		maxDelay = 8 * time.Second
	}

	delay := base << (attempt - 1)
	if delay <= 0 || delay > maxDelay {
		delay = maxDelay
	}

	if after := parseRetryAfter(retryAfterHeader, time.Now()); after > 0 {
		if after > maxRetryAfter {
			after = maxRetryAfter
		}
		if after > delay {
			delay = after
		}
	}

	// Full jitter in [delay/2, delay].
	if delay > 0 {
		half := delay / 2
		delay = half + time.Duration(rand.Int64N(int64(half)+1))
	}
	return delay
}

// parseRetryAfter parses the Retry-After header as either seconds or an HTTP
// date. It returns 0 when the header is absent or invalid.
func parseRetryAfter(header string, now time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		if d := when.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

// sleepOrDone waits for d or until the context is cancelled.
func sleepOrDone(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
