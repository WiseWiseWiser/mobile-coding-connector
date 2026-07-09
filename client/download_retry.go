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
	defaultDownloadMaxAttempts = 5
	defaultDownloadRetryBaseMs = 500
)

// DownloadRetryConfig configures per-request download retries.
type DownloadRetryConfig struct {
	MaxAttempts int
	// Backoff returns delay in milliseconds before the next attempt after a
	// failed try. Zero means no wait. Nil uses exponential default.
	Backoff func(attempt int) int64
}

// DownloadOptions configures optional download behavior.
type DownloadOptions struct {
	Retry  *DownloadRetryConfig
	DryRun bool
}

type downloadHTTPError struct {
	statusCode int
	status     string
	body       string
}

func (e *downloadHTTPError) Error() string {
	if e.body != "" {
		return fmt.Sprintf("%s: %s", e.status, e.body)
	}
	return e.status
}

type resolvedDownloadRetry struct {
	maxAttempts int
	backoff     func(attempt int) time.Duration
}

func (opts DownloadOptions) resolvedDownloadRetry() resolvedDownloadRetry {
	max := defaultDownloadMaxAttempts
	backoff := defaultDownloadBackoff
	if opts.Retry != nil {
		max = opts.Retry.MaxAttempts
		if max < 1 {
			max = defaultDownloadMaxAttempts
		}
		if opts.Retry.Backoff != nil {
			fn := opts.Retry.Backoff
			backoff = func(attempt int) time.Duration {
				return time.Duration(fn(attempt)) * time.Millisecond
			}
		}
	}
	if max < 1 {
		max = 1
	}
	return resolvedDownloadRetry{maxAttempts: max, backoff: backoff}
}

func defaultDownloadBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := defaultDownloadRetryBaseMs << (attempt - 1)
	return time.Duration(delay) * time.Millisecond
}

// IsRetryableDownloadError reports whether a download error warrants retry.
func IsRetryableDownloadError(err error) bool {
	var he *downloadHTTPError
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

func readDownloadAPIError(resp *http.Response) error {
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
	return &downloadHTTPError{
		statusCode: resp.StatusCode,
		status:     resp.Status,
		body:       body,
	}
}

func (c *Client) downloadGETWithRetry(
	resolvedRemote string,
	localPath string,
	startOffset int64,
	truncate bool,
	remoteSize int64,
	opts DownloadOptions,
	onProgress func(DownloadProgress),
) (int64, error) {
	cfg := opts.resolvedDownloadRetry()
	var lastErr error
	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		currentOffset := startOffset
		if attempt > 1 {
			if st, err := osStatRegular(localPath); err == nil {
				currentOffset = st
			}
		}
		shouldTruncate := truncate && attempt == 1 && currentOffset == 0
		written, err := c.downloadGETOnce(resolvedRemote, localPath, currentOffset, shouldTruncate, remoteSize, onProgress)
		if err == nil {
			return written, nil
		}
		lastErr = err
		if !IsRetryableDownloadError(err) {
			return 0, err
		}
		if attempt < cfg.maxAttempts {
			if onProgress != nil {
				onProgress(DownloadProgress{
					CompletedBytes: currentOffset,
					TotalBytes:     remoteSize,
					Phase:          DownloadPhaseRetrying,
					Attempt:        attempt + 1,
					MaxAttempts:    cfg.maxAttempts,
					Err:            err,
				})
			}
			time.Sleep(cfg.backoff(attempt))
		}
	}
	return 0, lastErr
}