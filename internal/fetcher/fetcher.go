package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// Fetcher handles HTTP requests with retry logic and custom headers.
//
// This structure wraps the retryablehttp client to provide automatic retries
// for transient network failures, which are common when downloading multiple
// segments from streaming servers.
//
// See: https://context7.com/golang/go for Go HTTP client documentation
type Fetcher struct {
	client    *retryablehttp.Client
	userAgent string
}

// Options configures the Fetcher behavior.
type Options struct {
	// UserAgent sets the User-Agent header for requests
	UserAgent string
	// MaxRetries sets the maximum number of retry attempts
	MaxRetries int
	// RetryWaitMin is the minimum time to wait between retries
	RetryWaitMin time.Duration
	// RetryWaitMax is the maximum time to wait between retries
	RetryWaitMax time.Duration
}

// DefaultOptions returns sensible default options for the Fetcher.
func DefaultOptions() Options {
	return Options{
		UserAgent:    "m3u8dl/1.0",
		MaxRetries:   3,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 30 * time.Second,
	}
}

// New creates a new Fetcher with the given options.
//
// The Fetcher uses exponential backoff for retries and will automatically
// retry on network errors and 5xx server errors.
//
// See: https://context7.com/golang/go for Go documentation
func New(opts Options) *Fetcher {
	client := retryablehttp.NewClient()
	client.RetryMax = opts.MaxRetries
	client.RetryWaitMin = opts.RetryWaitMin
	client.RetryWaitMax = opts.RetryWaitMax
	client.Logger = nil // Disable default logging

	return &Fetcher{
		client:    client,
		userAgent: opts.UserAgent,
	}
}

// Fetch downloads content from the given URL.
//
// This method will automatically retry failed requests up to MaxRetries times
// with exponential backoff. It returns the response body as a byte slice.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - url: The URL to fetch
//
// Returns the fetched content and any error encountered.
//
// See: https://context7.com/golang/go for Go context documentation
func (f *Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	if f.userAgent != "" {
		req.Header.Set("User-Agent", f.userAgent)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	return body, nil
}

// FetchWithCallback downloads content and calls a callback with progress information.
//
// This is useful for large downloads where you want to track progress or provide
// user feedback.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - url: The URL to fetch
//   - callback: Function called with downloaded bytes count
//
// Returns the fetched content and any error encountered.
func (f *Fetcher) FetchWithCallback(ctx context.Context, url string, callback func(bytesRead int64)) ([]byte, error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	if f.userAgent != "" {
		req.Header.Set("User-Agent", f.userAgent)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	// Read with progress tracking
	var body []byte
	buf := make([]byte, 32*1024) // 32KB buffer
	totalRead := int64(0)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
			totalRead += int64(n)
			if callback != nil {
				callback(totalRead)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read response body from %s: %w", url, err)
		}
	}

	return body, nil
}
