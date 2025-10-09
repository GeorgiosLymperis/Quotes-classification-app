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
	http *http.Client
	userAgent string
	retries int
	baseBackoff time.Duration
	maxBackoff time.Duration
}

func NewClient(timeout time.Duration, userAgent string, retries int, baseBackoff, maxBackoff time.Duration) *Client {
	transport := &http.Transport{
		MaxIdleConns: 100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout: 90*time.Second,
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout: 10*time.Second,
			KeepAlive: 30*time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10*time.Second,
		ExpectContinueTimeout: 1*time.Second,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &Client{
		http: &http.Client{
			Timeout: timeout,
			Transport: transport,
		},
		userAgent: userAgent,
		retries: retries,
		baseBackoff: baseBackoff,
		maxBackoff: maxBackoff,
	}
}

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

func (c *Client) GetWithRetry(ctx context.Context, url string) ([]byte, error) {
	backoff := c.baseBackoff
	attempts := c.retries + 1
	for i := 0; i <= attempts; i++ {
		request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {return nil, err}
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
		jt := time.Duration(rand.Int63n(int64(backoff/2)))
		sleep := min(backoff + jt, c.maxBackoff)

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


func isRetryableStatus(code int) bool {
	// 5xx και 429 => retry
	return code == 429 || (code >= 500 && code <= 599)
}

func parseRetryAfter(ra string) time.Duration {
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

func sleepCtx(ctx context.Context, dur time.Duration) error {
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <- ctx.Done():
		return ctx.Err()
	case <- t.C:
		return nil
	}
}