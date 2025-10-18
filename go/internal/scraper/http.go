package scraper

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	http        *http.Client  // Underlying HTTP client.
	userAgent   string        // Optional User-Agent header to send on requests.
	retries     int           // Number of retry attempts for GetWithRetry (not counting the first try).
	baseBackoff time.Duration // Initial backoff duration before applying jitter/exponential growth.
	maxBackoff  time.Duration // Upper bound for backoff duration between retries.
}

// NewClient constructs a Client with sane Transport defaults and
// timeouts suitable for scraping workloads.
//
// Parameters:
//   - timeout: per-request deadline enforced by the underlying http.Client.
//   - userAgent: value for the "User-Agent" header (empty string disables it).
//   - retries: number of retry attempts performed by GetWithRetry (in addition to the first try).
//   - baseBackoff: initial backoff duration used by GetWithRetry.
//   - maxBackoff: maximum backoff cap used by GetWithRetry.
//
// The returned Client uses an http.Transport with connection pooling, TLS >= 1.2,
// and reasonable dial/handshake timeouts.
func NewClient(timeout time.Duration, userAgent string, retries int, baseBackoff, maxBackoff time.Duration) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &Client{
		http: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		userAgent:   userAgent,
		retries:     retries,
		baseBackoff: baseBackoff,
		maxBackoff:  maxBackoff,
	}
}

// Get performs a simple HTTP GET and returns the response body if the
// status code is 2xx. Non-2xx responses return an error whose message
// is the HTTP status line (e.g., "404 Not Found").
func (c *Client) Get(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New(response.Status)
	}
	return io.ReadAll(response.Body)
}

// GetWithRetry performs an HTTP GET with retries, jittered exponential
// backoff, and support for "Retry-After" when present.
//
// Behavior:
//   - Retries on 5xx and 429 responses (see isRetryableStatus).
//   - Honors "Retry-After" header (seconds form) before applying backoff.
//   - Adds jitter up to 50% of the current backoff.
//   - Caps backoff at maxBackoff.
//   - Aborts early if the provided context is canceled.
//
// Returns the response body for a 2xx status. For a non-retryable non-2xx
// status, it returns the body (best effort) and an error. If retries are
// exhausted without a successful 2xx response, it returns nil, nil.
func (c *Client) GetWithRetry(ctx context.Context, url string) ([]byte, error) {
	backoff := c.baseBackoff
	attempts := c.retries + 1
	for i := 0; i <= attempts; i++ {
		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		if c.userAgent != "" {
			request.Header.Set("User-Agent", c.userAgent)
		}

		response, err := c.http.Do(request)
		if err == nil && response != nil {
			defer response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return io.ReadAll(response.Body)
			}
			if !isRetryableStatus(response.StatusCode) {
				b, _ := io.ReadAll(response.Body)
				return b, errors.New(response.Status)
			}
			if ra := response.Header.Get("Retry-After"); ra != "" {
				if duration := parseRetryAfter(ra); duration > 0 {
					sleepCtx(ctx, duration)
				}
			}
		}
		jt := time.Duration(rand.Int63n(int64(backoff / 2)))
		sleep := min(backoff+jt, c.maxBackoff)

		if i < attempts-1 { // μην κοιμηθείς μετά το τελευταίο attempt
			if err := sleepCtx(ctx, sleep); err != nil {
				return nil, err
			}
		}
		// exponential
		if backoff < c.maxBackoff {
			backoff *= 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
		}
	}
	return nil, nil
}

// isRetryableStatus reports whether an HTTP status code should be retried.
// It returns true for 429 (Too Many Requests) and 5xx server errors.
func isRetryableStatus(code int) bool {
	return code == 429 || (code >= 500 && code <= 599)
}

// parseRetryAfter parses a Retry-After header expressed in seconds.
// Returns 0 if parsing fails or if the value is non-positive.
func parseRetryAfter(ra string) time.Duration {
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// sleepCtx sleeps for the given duration or returns early if the context is canceled.
func sleepCtx(ctx context.Context, dur time.Duration) error {
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
