// Package attachtest is the L2 harness for tests/terminal-cli-attach.
package attachtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/cmd/agentcli"
	"github.com/xhd2015/doctest/session"
	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap"
	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
)

// Request is the doctest harness request.
type Request struct {
	Phase  string
	Marker string
}

// Response is the doctest harness response.
type Response struct {
	SessionID         string
	AttachOutput      string
	MarkerInOutput    bool
	HasAltScreenReset bool
	SessionStatus     string
	WriterConnected   bool
	AttachErr         string
	DidAttach         bool
}

// Run executes a terminal-cli-attach phase.
func Run(t *testing.T, _ *session.Doctest, req *Request) (*Response, error) {
	switch req.Phase {
	case "writer-held-echo":
		return runWriterHeldEcho(t, req)
	case "exited-refuse":
		return runExitedRefuse(t, req)
	default:
		return nil, fmt.Errorf("unknown phase %q", req.Phase)
	}
}

func runWriterHeldEcho(t *testing.T, req *Request) (*Response, error) {
	marker := strings.TrimSpace(req.Marker)
	if marker == "" {
		marker = "CLI_ATTACH_MARKER"
	}

	mux := http.NewServeMux()
	ptywrap.RegisterAPIWithManager(mux, ptywrap.NewManager())
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	sessionID, err := createShellSession(srv.URL)
	if err != nil {
		return nil, err
	}
	writer, err := holdWriter(srv.URL, sessionID)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = writer.Close() })

	stdinR, stdinW := io.Pipe()
	t.Cleanup(func() { _ = stdinR.Close(); _ = stdinW.Close() })

	var stdout syncBuffer
	errCh := make(chan error, 1)
	go func() {
		opts := agentcli.TerminalAttachConnectOptions(sessionID)
		opts.SkipTTYCheck = true
		_, aerr := ptyclient.AttachWithIO(ptyclient.NewClient(srv.URL), opts, stdinR, &stdout, io.Discard)
		errCh <- aerr
	}()

	// In-place attach sends no snapshot; do not wait for output before typing.
	time.Sleep(400 * time.Millisecond)

	if _, werr := io.WriteString(stdinW, "echo "+marker+"\n"); werr != nil {
		return nil, fmt.Errorf("write input: %w", werr)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(stdout.String(), marker) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = stdinW.Close()
	select {
	case aerr := <-errCh:
		if aerr != nil {
			// stdin close ends the attach Wait; that is not a product failure.
		}
	case <-time.After(2 * time.Second):
	}

	info, lerr := getSession(srv.URL, sessionID)
	if lerr != nil {
		return nil, lerr
	}

	out := stdout.String()
	return &Response{
		SessionID:         sessionID,
		AttachOutput:      out,
		MarkerInOutput:    strings.Contains(out, marker),
		HasAltScreenReset: strings.Contains(out, "\x1b[?1049l") || strings.Contains(out, "\x1b[2J"),
		SessionStatus:     info.Status,
		WriterConnected:   info.WriterConnected,
		DidAttach:         true,
	}, nil
}

func runExitedRefuse(t *testing.T, _ *Request) (*Response, error) {
	mux := http.NewServeMux()
	ptywrap.RegisterAPIWithManager(mux, ptywrap.NewManager())
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	sessionID, err := createShellSession(srv.URL)
	if err != nil {
		return nil, err
	}
	writer, err := holdWriter(srv.URL, sessionID)
	if err != nil {
		return nil, err
	}
	_ = writer.Close()

	var info ptywrap.SessionInfo
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		info, err = getSession(srv.URL, sessionID)
		if err == nil && info.Status == "exited" {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if err != nil {
		return nil, err
	}

	gateErr := agentcli.ErrIfSessionNotAttachable(info)
	resp := &Response{
		SessionID:       sessionID,
		SessionStatus:   info.Status,
		WriterConnected: info.WriterConnected,
		DidAttach:       false,
	}
	if gateErr != nil {
		resp.AttachErr = gateErr.Error()
	}
	return resp, nil
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func createShellSession(base string) (string, error) {
	resp, err := http.Post(base+"/api/terminal/sessions", "application/json", strings.NewReader(`{"name":"web-writer"}`))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session: %s %s", resp.Status, raw)
	}
	var info ptywrap.SessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	if info.ID == "" {
		return "", fmt.Errorf("create session: empty id")
	}
	return info.ID, nil
}

func holdWriter(base, sessionID string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(base, "http") + "/api/terminal?session_id=" + sessionID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hold writer: %w", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 4; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	_ = conn.SetReadDeadline(time.Time{})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	return conn, nil
}

func getSession(base, id string) (ptywrap.SessionInfo, error) {
	resp, err := http.Get(base + "/api/terminal/sessions?page=1&page_size=100")
	if err != nil {
		return ptywrap.SessionInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return ptywrap.SessionInfo{}, fmt.Errorf("list sessions: %s %s", resp.Status, raw)
	}
	var page ptywrap.SessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return ptywrap.SessionInfo{}, err
	}
	for _, s := range page.Sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return ptywrap.SessionInfo{}, fmt.Errorf("session %s not in list", id)
}
