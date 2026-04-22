// Package exec exposes server-side command execution endpoints for the
// remote-agent CLI:
//
//   - POST /api/exec streams stdout/stderr as newline-delimited JSON (NDJSON)
//     and is used for non-interactive/scripted execution.
//   - GET /api/exec/ws upgrades to a WebSocket-backed PTY session and is used
//     when `remote-agent exec` is launched from an interactive terminal.
//
// Both endpoints resolve binaries via tool_exec so the server's configured
// PATH extensions apply the same way they do for other server-managed
// processes.
//
// NDJSON event protocol (one JSON object per line):
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
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

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

type execControlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

type wsConnWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsConnWriter) writeMessage(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(messageType, data)
}

func (w *wsConnWriter) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.writeMessage(websocket.TextMessage, data)
}

var execWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RegisterAPI registers the /api/exec endpoint.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/exec", handleExec)
	mux.HandleFunc("/api/exec/ws", handleExecWebSocket)
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

func handleExecWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := execWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	req, err := readExecStartMessage(conn)
	if err != nil {
		_ = writeExecWSError(conn, err.Error())
		return
	}

	prepared, err := tool_exec.New(req.Argv[0], req.Argv[1:], nil)
	if err != nil {
		_ = writeExecWSError(conn, fmt.Sprintf("failed to resolve command: %v", err))
		return
	}

	ctxCmd := exec.CommandContext(r.Context(), prepared.Path, prepared.Args[1:]...)
	ctxCmd.Env = prepared.Env
	ctxCmd.Dir = prepared.Dir

	ptmx, err := pty.StartWithSize(ctxCmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		_ = writeExecWSError(conn, fmt.Sprintf("failed to start: %v", err))
		return
	}
	defer ptmx.Close()

	writer := &wsConnWriter{conn: conn}

	stdinErrCh := make(chan error, 1)
	go func() {
		stdinErrCh <- forwardExecInput(conn, ptmx)
	}()

	outputErrCh := make(chan error, 1)
	go func() {
		outputErrCh <- pumpExecPTY(ptmx, writer)
	}()

	waitErrCh := make(chan error, 1)
	go func() {
		waitErrCh <- ctxCmd.Wait()
	}()

	select {
	case waitErr := <-waitErrCh:
		exitCode, err := execExitCode(waitErr)
		if err != nil {
			_ = writer.writeJSON(map[string]any{"type": "error", "message": fmt.Sprintf("wait: %v", err)})
			return
		}
		if outputErr := <-outputErrCh; outputErr != nil {
			return
		}
		_ = writer.writeJSON(map[string]any{"type": "exit", "code": exitCode})
	case <-stdinErrCh:
		killProcess(ctxCmd)
		<-waitErrCh
	case err := <-outputErrCh:
		if err == nil {
			waitErr := <-waitErrCh
			exitCode, exitErr := execExitCode(waitErr)
			if exitErr != nil {
				_ = writer.writeJSON(map[string]any{"type": "error", "message": fmt.Sprintf("wait: %v", exitErr)})
				return
			}
			_ = writer.writeJSON(map[string]any{"type": "exit", "code": exitCode})
			return
		}
		killProcess(ctxCmd)
		<-waitErrCh
	}
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

func readExecStartMessage(conn *websocket.Conn) (*ExecRequest, error) {
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("expected exec start request")
	}

	var req ExecRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid exec start request: %w", err)
	}
	if len(req.Argv) == 0 {
		return nil, fmt.Errorf("argv must contain at least the binary name")
	}
	return &req, nil
}

func writeExecWSError(conn *websocket.Conn, message string) error {
	return conn.WriteJSON(map[string]any{"type": "error", "message": message})
}

func forwardExecInput(conn *websocket.Conn, ptmx *os.File) error {
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if msgType == websocket.TextMessage {
			var msg execControlMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
				if msg.Cols > 0 && msg.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(msg.Rows),
						Cols: uint16(msg.Cols),
					})
				}
				continue
			}
		}
		if _, err := ptmx.Write(data); err != nil {
			return err
		}
	}
}

func pumpExecPTY(ptmx *os.File, writer *wsConnWriter) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			if writeErr := writer.writeMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func execExitCode(waitErr error) (int, error) {
	if waitErr == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, waitErr
}

func killProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
