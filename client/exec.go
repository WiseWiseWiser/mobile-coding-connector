package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// ExecRequest is the body of POST /api/exec. Kept in sync with the server's
// server/exec.ExecRequest.
type ExecRequest struct {
	Argv []string `json:"argv"`
}

// ExecEvent is one NDJSON record returned by POST /api/exec.
//
// Type is one of:
//   - "stdout"    — Data carries a chunk of the remote process's stdout.
//   - "stderr"    — Data carries a chunk of the remote process's stderr.
//   - "heartbeat" — Keep-alive ping emitted during idle gaps. Safe to ignore.
//   - "exit"      — Code carries the remote process's exit code; stream ends.
//   - "error"     — Message carries a server-side error; stream ends.
type ExecEvent struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ExecHandler receives streamed events. It may be called concurrently is NOT
// a concern -- the client consumes the response serially.
type ExecHandler func(ExecEvent)

// Exec runs argv as a subprocess on the server and streams its stdout/stderr
// back via handler. The returned exit code mirrors the remote process's exit
// code on success. A non-zero remote exit code is NOT returned as a Go
// error; inspect the returned code. An error is returned only when the HTTP
// call itself failed, the server refused the request, the stream was
// truncated, or the server emitted an "error" event.
func (c *Client) Exec(req ExecRequest, handler ExecHandler) (int, error) {
	if len(req.Argv) == 0 {
		return 0, fmt.Errorf("exec: argv must contain at least the binary name")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("exec: marshal request: %w", err)
	}

	httpReq, err := c.NewRequest(http.MethodPost, "/api/exec", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, readAPIError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Chunks can be large (32 KiB on the server), so lift the default limit.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var (
		exitCode   int
		sawExit    bool
		streamErr  error
	)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev ExecEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return 0, fmt.Errorf("exec: decode event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		switch ev.Type {
		case "exit":
			exitCode = ev.Code
			sawExit = true
		case "error":
			streamErr = fmt.Errorf("server error: %s", ev.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("exec: read stream: %w", err)
	}
	if streamErr != nil {
		return 0, streamErr
	}
	if !sawExit {
		return 0, fmt.Errorf("exec: stream ended without exit event")
	}
	return exitCode, nil
}
