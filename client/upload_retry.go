package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultChunkMaxAttempts = 5
	defaultChunkRetryBaseMs = 500
)

// ChunkRetryConfig configures per-chunk upload retries.
type ChunkRetryConfig struct {
	MaxAttempts int
	// Backoff returns delay in milliseconds before the next attempt after a
	// failed try. Zero means no wait. Nil uses exponential default.
	Backoff func(attempt int) int64
}

type uploadHTTPError struct {
	statusCode int
	status     string
	body       string
}

func (e *uploadHTTPError) Error() string {
	if e.body != "" {
		return fmt.Sprintf("%s: %s", e.status, e.body)
	}
	return e.status
}

type resolvedChunkRetry struct {
	maxAttempts int
	backoff     func(attempt int) time.Duration
}

func (opts UploadOptions) resolvedChunkRetry() resolvedChunkRetry {
	max := defaultChunkMaxAttempts
	backoff := defaultChunkBackoff
	if opts.ChunkRetry != nil {
		// MaxAttempts >= 1 is honored as-is (1 = no retry). Zero or negative
		// falls back to the production default.
		max = opts.ChunkRetry.MaxAttempts
		if max < 1 {
			max = defaultChunkMaxAttempts
		}
		if opts.ChunkRetry.Backoff != nil {
			fn := opts.ChunkRetry.Backoff
			backoff = func(attempt int) time.Duration {
				return time.Duration(fn(attempt)) * time.Millisecond
			}
		}
	}
	if max < 1 {
		max = 1
	}
	return resolvedChunkRetry{maxAttempts: max, backoff: backoff}
}

func defaultChunkBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := defaultChunkRetryBaseMs << (attempt - 1)
	return time.Duration(delay) * time.Millisecond
}

// IsRetryableUploadError reports whether a chunk upload error warrants retry.
func IsRetryableUploadError(err error) bool {
	var he *uploadHTTPError
	if errors.As(err, &he) {
		switch he.statusCode {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		if he.statusCode >= 500 {
			return true
		}
		if he.statusCode >= 400 && he.statusCode < 500 {
			return false
		}
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "reset") ||
		strings.Contains(msg, "eof") ||
		strings.Contains(msg, "broken pipe")
}

func readUploadAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	body := strings.TrimSpace(string(data))
	if json.Unmarshal(data, &errResp) == nil && errResp.Error != "" {
		body = errResp.Error
	}
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	return &uploadHTTPError{
		statusCode: resp.StatusCode,
		status:     resp.Status,
		body:       body,
	}
}

func (c *Client) uploadChunkWithRetry(uploadID string, chunkIndex int, chunk []byte, opts UploadOptions) error {
	cfg := opts.resolvedChunkRetry()
	var lastErr error
	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ { // maxAttempts is always >= 1
		lastErr = c.uploadChunk(uploadID, chunkIndex, chunk)
		if lastErr == nil {
			return nil
		}
		if !IsRetryableUploadError(lastErr) {
			return lastErr
		}
		if attempt < cfg.maxAttempts {
			time.Sleep(cfg.backoff(attempt))
		}
	}
	return lastErr
}