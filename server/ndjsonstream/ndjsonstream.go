// Package ndjsonstream provides a small utility for writing newline-delimited
// JSON (NDJSON) event streams over HTTP, with a heartbeat loop that fills
// idle gaps so reverse proxies (e.g. Cloudflare tunnels) don't drop the
// connection during long-running commands.
//
// It is used by /api/exec and /api/git/clone, both of which stream
// stdout/stderr chunks from a subprocess back to the client.
package ndjsonstream

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HeartbeatInterval is the maximum idle gap allowed between events on an
// NDJSON stream before a heartbeat is emitted. It is chosen below common
// proxy idle timeouts (Cloudflare tunnel: 100s).
const HeartbeatInterval = 60 * time.Second

// Writer serializes JSON events to an HTTP response as NDJSON and tracks
// the time of the last send so a heartbeat goroutine can fill idle gaps.
type Writer struct {
	mu       sync.Mutex
	w        http.ResponseWriter
	flusher  http.Flusher
	lastSend time.Time
}

// NewWriter prepares the response headers for NDJSON streaming and returns
// a new Writer bound to w. It MUST be called before the first event is
// written.
func NewWriter(w http.ResponseWriter) *Writer {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
	return &Writer{w: w, flusher: flusher, lastSend: time.Now()}
}

// Send marshals v and writes it as a single NDJSON line, flushing the
// underlying response so the client sees the event immediately.
func (n *Writer) Send(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.w.Write(data)
	n.w.Write([]byte{'\n'})
	if n.flusher != nil {
		n.flusher.Flush()
	}
	n.lastSend = time.Now()
}

// SendError emits an {"type":"error","message":...} event.
func (n *Writer) SendError(message string) {
	n.Send(map[string]any{"type": "error", "message": message})
}

// sendHeartbeatIfIdle emits a heartbeat only when the stream has been idle
// for at least minIdle.
func (n *Writer) sendHeartbeatIfIdle(minIdle time.Duration) {
	n.mu.Lock()
	idle := time.Since(n.lastSend) >= minIdle
	n.mu.Unlock()
	if !idle {
		return
	}
	n.Send(map[string]any{"type": "heartbeat"})
}

// RunHeartbeat ticks every interval/2 and emits a heartbeat whenever the
// stream has been idle for at least interval. It stops when stop is closed.
// Intended to be launched as a goroutine for the duration of a request.
func RunHeartbeat(stream *Writer, interval time.Duration, stop <-chan struct{}) {
	tick := interval / 2
	if tick <= 0 {
		tick = interval
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			stream.sendHeartbeatIfIdle(interval)
		}
	}
}
