// Package browser streams a headed Chrome window to the viewer and injects the
// viewer's mouse and keyboard input back into that window.
//
// Frames come from the browser-agent extension over the Chrome DevTools
// Protocol (Page.captureScreenshot), driven through the local browser-agent
// daemon. Using the real browser profile has three consequences that make this
// the only workable remote-control path while the Mac is locked: the page keeps
// rendering, the user stays logged in, and Chrome's own renderer applies the
// input instead of the window server.
package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DaemonPort is the fixed browser-agent daemon port. The extension manifest
	// hardcodes it, so ai-critic always talks to that port instead of picking
	// its own.
	DaemonPort = 43761
	// SessionID is reused for every run. browser-agent never reaps idle
	// sessions, so a fresh id per launch would leak one session row and one
	// registered Chrome window each time.
	SessionID = "ai-critic"
)

// daemon is a minimal HTTP client for the browser-agent daemon.
type daemon struct {
	base   string
	client *http.Client
}

func newDaemon(port int) *daemon {
	return &daemon{
		base: fmt.Sprintf("http://127.0.0.1:%d", port),
		// Per-request deadlines carry the real timeout; a global client timeout
		// would cut off slow screenshot jobs on large windows.
		client: &http.Client{},
	}
}

type daemonHealth struct {
	OK      bool   `json:"ok"`
	Version string `json:"daemon_version"`
	BaseDir string `json:"base_dir"`
	Product string `json:"product"`
}

type extensionInfo struct {
	Connected bool   `json:"connected"`
	Version   string `json:"version"`
}

type attachInfo struct {
	LastError string `json:"last_error"`
}

// sessionRow is one entry of GET /v1/sessions (a bare JSON array).
type sessionRow struct {
	SessionID        string        `json:"session_id"`
	Phase            string        `json:"phase"`
	Extension        extensionInfo `json:"extension"`
	SessionPageCount int           `json:"session_page_count"`
	Attach           attachInfo    `json:"attach"`
	// ExtensionInstallPath is the unpacked directory the session page expects;
	// Chrome derives its extension id from this absolute path.
	ExtensionInstallPath string `json:"extension_install_path,omitempty"`
}

type jobResponse struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error"`
	Data  json.RawMessage `json:"data"`
}

func (d *daemon) health(ctx context.Context) (*daemonHealth, error) {
	var h daemonHealth
	if err := d.getJSON(ctx, "/v1/health", &h); err != nil {
		return nil, err
	}
	if !h.OK {
		return nil, fmt.Errorf("daemon reports not ok")
	}
	return &h, nil
}

func (d *daemon) sessions(ctx context.Context) ([]sessionRow, error) {
	body, err := d.do(ctx, http.MethodGet, "/v1/sessions", nil)
	if err != nil {
		return nil, err
	}
	// The daemon returns a bare array; accept a wrapped object too so a future
	// protocol change does not break the page.
	var rows []sessionRow
	if err := json.Unmarshal(body, &rows); err == nil {
		return rows, nil
	}
	var wrapped struct {
		Sessions []sessionRow `json:"sessions"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode sessions: %w", err)
	}
	return wrapped.Sessions, nil
}

func (d *daemon) session(ctx context.Context, id string) (*sessionRow, error) {
	var row sessionRow
	if err := d.getJSON(ctx, "/v1/session?session="+url.QueryEscape(id), &row); err != nil {
		return nil, err
	}
	return &row, nil
}

// createSession registers (or reuses) a session id. An already-registered id is
// success: the daemon answers 409 for it.
func (d *daemon) createSession(ctx context.Context, id string) error {
	body, err := json.Marshal(map[string]any{"session_id": id})
	if err != nil {
		return err
	}
	resp, raw, err := d.request(ctx, http.MethodPost, "/v1/sessions", body)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
		return nil
	}
	// The daemon also reports "already exists" as a 4xx JSON error.
	if strings.Contains(strings.ToLower(string(raw)), "already") {
		return nil
	}
	return fmt.Errorf("create session: status %d: %s", resp.StatusCode, truncate(raw, 200))
}

// job runs one extension job and returns its data payload. Browser-agent
// answers synchronously; a non-ok body is an error.
//
// The session is explicit because one Chrome window hosts one session: two
// windows means two sessions, and every job (screenshot, input, tab list) must
// be addressed to the window whose page the viewer is looking at.
func (d *daemon) job(ctx context.Context, sessionID string, tabID int64, typ string, params any, timeout time.Duration) (json.RawMessage, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if sessionID == "" {
		sessionID = SessionID
	}
	payload := map[string]any{
		"session_id": sessionID,
		"type":       typ,
		"timeout_ms": int(timeout / time.Millisecond),
	}
	if tabID != 0 {
		payload["tab_id"] = tabID
	}
	if params != nil {
		payload["params"] = params
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	jobCtx, cancel := context.WithTimeout(ctx, timeout+10*time.Second)
	defer cancel()
	resp, raw, err := d.request(jobCtx, http.MethodPost, "/v1/jobs", body)
	if err != nil {
		return nil, err
	}
	var out jobResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode job %s: %w (status %d)", typ, err, resp.StatusCode)
	}
	if !out.OK {
		msg := out.Error
		if msg == "" {
			msg = fmt.Sprintf("job %s failed (status %d)", typ, resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return out.Data, nil
}

func (d *daemon) getJSON(ctx context.Context, path string, v any) error {
	body, err := d.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (d *daemon) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	_, raw, err := d.request(ctx, method, path, body)
	return raw, err
}

func (d *daemon) request(ctx context.Context, method, path string, body []byte) (*http.Response, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, d.base+path, reader)
	if err != nil {
		return nil, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return resp, nil, err
	}
	if resp.StatusCode >= 400 && len(raw) == 0 {
		return resp, raw, fmt.Errorf("%s %s: status %d", method, path, resp.StatusCode)
	}
	return resp, raw, nil
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
