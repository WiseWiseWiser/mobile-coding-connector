package client

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
)

// ChunkTransportTracker records per-chunk HTTP RoundTrip attempts (including
// transport-level failures before the request reaches the server).
type ChunkTransportTracker struct {
	Attempts map[int]int
}

// FlakyChunkTransportConfig injects transport-level failures on chunk uploads.
type FlakyChunkTransportConfig struct {
	FailChunkIndex int
	FailCount      int
	FailErr        error
	Tracker        *ChunkTransportTracker
}

// WrapFlakyChunkTransport returns an http.RoundTripper that fails the first
// FailCount chunk POSTs for FailChunkIndex with FailErr, then delegates to base.
func WrapFlakyChunkTransport(base http.RoundTripper, cfg FlakyChunkTransportConfig) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if cfg.FailErr == nil {
		cfg.FailErr = fmt.Errorf("connection reset by peer")
	}
	if cfg.Tracker == nil {
		cfg.Tracker = &ChunkTransportTracker{Attempts: make(map[int]int)}
	}
	if cfg.Tracker.Attempts == nil {
		cfg.Tracker.Attempts = make(map[int]int)
	}
	failCounts := make(map[int]int)
	return flakyChunkTransport{base: base, cfg: cfg, failCounts: failCounts}
}

type flakyChunkTransport struct {
	base       http.RoundTripper
	cfg        FlakyChunkTransportConfig
	failCounts map[int]int
}

func (t flakyChunkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/api/files/upload/chunk" && req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		if idx, ok := parseChunkIndexFromMultipart(body, req.Header.Get("Content-Type")); ok {
			t.cfg.Tracker.Attempts[idx]++
			if t.cfg.FailCount > 0 && idx == t.cfg.FailChunkIndex {
				if t.failCounts[idx] < t.cfg.FailCount {
					t.failCounts[idx]++
					return nil, t.cfg.FailErr
				}
			}
		}
	}
	return t.base.RoundTrip(req)
}

func parseChunkIndexFromMultipart(body []byte, contentType string) (int, bool) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return 0, false
	}
	boundary := params["boundary"]
	if boundary == "" {
		return 0, false
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, false
		}
		if part.FormName() == "chunk_index" {
			val, err := io.ReadAll(part)
			part.Close()
			if err != nil {
				return 0, false
			}
			idx, err := strconv.Atoi(string(val))
			return idx, err == nil
		}
		part.Close()
	}
	return 0, false
}