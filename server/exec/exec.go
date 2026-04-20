// Package exec exposes an HTTP endpoint that runs a subprocess on the server
// and streams its stdout and stderr back to the client as newline-delimited
// JSON (NDJSON) events, followed by a final "exit" event carrying the exit
// code.
//
// It is used by the `remote-agent exec` CLI subcommand to run ad-hoc commands
// on the server host. The command's PATH is augmented with the server's
// configured extra paths (via tool_resolve/tool_exec), so user-installed tools
// are discoverable the same way they are for other server-managed processes.
//
// Event protocol (one JSON object per line):
//
//	{"type":"stdout","data":"..."}
//	{"type":"stderr","data":"..."}
//	{"type":"heartbeat"}
//	{"type":"exit","code":N}
//	{"type":"error","message":"..."}
//
// `data` chunks are UTF-8 strings; non-UTF-8 bytes are replaced. Heartbeat
// events are emitted when no other event has been sent for
// HeartbeatInterval, to keep intermediaries (e.g. Cloudflare tunnels) from
// closing the connection on idle timeouts. Clients must accept (and may
// ignore) heartbeat events.
package exec

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

// HeartbeatInterval is the maximum idle gap allowed between events on the
// /api/exec stream before a heartbeat is emitted. It is chosen below common
// proxy idle timeouts (Cloudflare tunnel: 100s).
const HeartbeatInterval = 60 * time.Second

// ExecRequest is the JSON body accepted by POST /api/exec.
type ExecRequest struct {
	// Argv is the command and its arguments. Argv[0] is the binary and the
	// rest are passed as arguments. The binary is resolved via tool_exec,
	// which honours the server's PATH extensions.
	Argv []string `json:"argv"`
}

// RegisterAPI registers the /api/exec endpoint.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/exec", handleExec)
}

func handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if len(req.Argv) == 0 {
		writeJSONError(w, http.StatusBadRequest, "argv must contain at least the binary name")
		return
	}

	prepared, err := tool_exec.New(req.Argv[0], req.Argv[1:], nil)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to resolve command: %v", err))
		return
	}

	// Bind the child process to the request context so a client
	// disconnection terminates it instead of leaving orphans.
	ctxCmd := exec.CommandContext(r.Context(), prepared.Path, prepared.Args[1:]...)
	ctxCmd.Env = prepared.Env
	ctxCmd.Dir = prepared.Dir

	// Switch response to NDJSON streaming BEFORE we start writing events.
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, _ := w.(http.Flusher)
	stream := newNDJSONWriter(w, flusher)

	// Heartbeat loop: emit a heartbeat event when the stream has been idle
	// for HeartbeatInterval, so reverse proxies (e.g. Cloudflare tunnels)
	// don't drop the connection while a long-running command produces no
	// output. Stops when the handler returns.
	stopHeartbeat := make(chan struct{})
	var heartbeatDone sync.WaitGroup
	heartbeatDone.Add(1)
	go func() {
		defer heartbeatDone.Done()
		runHeartbeat(stream, HeartbeatInterval, stopHeartbeat)
	}()
	defer func() {
		close(stopHeartbeat)
		heartbeatDone.Wait()
	}()

	stdoutPipe, err := ctxCmd.StdoutPipe()
	if err != nil {
		stream.sendError(fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	stderrPipe, err := ctxCmd.StderrPipe()
	if err != nil {
		stream.sendError(fmt.Sprintf("stderr pipe: %v", err))
		return
	}

	if err := ctxCmd.Start(); err != nil {
		stream.sendError(fmt.Sprintf("failed to start: %v", err))
		return
	}

	// Read stdout and stderr concurrently, forwarding each chunk to the
	// client as it arrives.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pumpPipe(stdoutPipe, "stdout", stream)
	}()
	go func() {
		defer wg.Done()
		pumpPipe(stderrPipe, "stderr", stream)
	}()
	wg.Wait()

	exitCode := 0
	waitErr := ctxCmd.Wait()
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			stream.sendError(fmt.Sprintf("wait: %v", waitErr))
			return
		}
	}

	stream.send(map[string]any{"type": "exit", "code": exitCode})
}

// pumpPipe reads from pipe and forwards each non-empty chunk as a typed event.
func pumpPipe(pipe io.Reader, kind string, stream *ndjsonWriter) {
	buf := make([]byte, 32*1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			stream.send(map[string]any{"type": kind, "data": safeString(buf[:n])})
		}
		if err != nil {
			return
		}
	}
}

// safeString returns s as a valid UTF-8 string, replacing invalid sequences
// with U+FFFD. This keeps the JSON encoder's output meaningful on binary
// output (and avoids silently dropping bytes).
func safeString(b []byte) string {
	return strings.ToValidUTF8(string(b), "\uFFFD")
}

// ndjsonWriter serializes writes so concurrent pumpers don't interleave
// bytes within a single JSON line. It also tracks the time of the last send
// so a heartbeat goroutine can fill idle gaps.
type ndjsonWriter struct {
	mu       sync.Mutex
	w        http.ResponseWriter
	flusher  http.Flusher
	lastSend time.Time
}

func newNDJSONWriter(w http.ResponseWriter, flusher http.Flusher) *ndjsonWriter {
	return &ndjsonWriter{w: w, flusher: flusher, lastSend: time.Now()}
}

func (n *ndjsonWriter) send(v any) {
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

// sendHeartbeatIfIdle emits a heartbeat only when the stream has been idle
// for at least minIdle. Returns true if a heartbeat was sent.
func (n *ndjsonWriter) sendHeartbeatIfIdle(minIdle time.Duration) bool {
	n.mu.Lock()
	idle := time.Since(n.lastSend) >= minIdle
	n.mu.Unlock()
	if !idle {
		return false
	}
	n.send(map[string]any{"type": "heartbeat"})
	return true
}

func (n *ndjsonWriter) sendError(message string) {
	n.send(map[string]any{"type": "error", "message": message})
}

// runHeartbeat wakes up frequently enough to guarantee that any idle gap of
// at least `interval` is covered by at most one missed tick. It stops when
// `stop` is closed.
func runHeartbeat(stream *ndjsonWriter, interval time.Duration, stop <-chan struct{}) {
	// Check twice per interval so we catch idleness near the boundary.
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

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
