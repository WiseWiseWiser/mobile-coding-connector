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
// ndjsonstream.HeartbeatInterval, to keep intermediaries (e.g. Cloudflare
// tunnels) from closing the connection on idle timeouts. Clients must
// accept (and may ignore) heartbeat events.
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

	"github.com/xhd2015/lifelog-private/ai-critic/server/ndjsonstream"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

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
	stream := ndjsonstream.NewWriter(w)

	// Heartbeat loop: emit a heartbeat event when the stream has been idle
	// for HeartbeatInterval, so reverse proxies (e.g. Cloudflare tunnels)
	// don't drop the connection while a long-running command produces no
	// output. Stops when the handler returns.
	stopHeartbeat := make(chan struct{})
	var heartbeatDone sync.WaitGroup
	heartbeatDone.Add(1)
	go func() {
		defer heartbeatDone.Done()
		ndjsonstream.RunHeartbeat(stream, ndjsonstream.HeartbeatInterval, stopHeartbeat)
	}()
	defer func() {
		close(stopHeartbeat)
		heartbeatDone.Wait()
	}()

	stdoutPipe, err := ctxCmd.StdoutPipe()
	if err != nil {
		stream.SendError(fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	stderrPipe, err := ctxCmd.StderrPipe()
	if err != nil {
		stream.SendError(fmt.Sprintf("stderr pipe: %v", err))
		return
	}

	if err := ctxCmd.Start(); err != nil {
		stream.SendError(fmt.Sprintf("failed to start: %v", err))
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
			stream.SendError(fmt.Sprintf("wait: %v", waitErr))
			return
		}
	}

	stream.Send(map[string]any{"type": "exit", "code": exitCode})
}

// pumpPipe reads from pipe and forwards each non-empty chunk as a typed event.
func pumpPipe(pipe io.Reader, kind string, stream *ndjsonstream.Writer) {
	buf := make([]byte, 32*1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			stream.Send(map[string]any{"type": kind, "data": safeString(buf[:n])})
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

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
