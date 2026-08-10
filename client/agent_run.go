package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// AgentRunSession is one entry from GET /api/agent-run/sessions.
type AgentRunSession struct {
	SessionID string `json:"session_id"`
	Runner    string `json:"runner"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ListAgentRunSessions calls GET /api/agent-run/sessions?limit=N.
// limit is always sent as a query param (0 = all, N > 0 = cap; server default
// applies only when the query is omitted, which this client does not do).
// total is the full count before the limit is applied.
func (c *Client) ListAgentRunSessions(limit int) (sessions []AgentRunSession, total int, err error) {
	path := "/api/agent-run/sessions?limit=" + url.QueryEscape(strconv.Itoa(limit))
	var out struct {
		Sessions []AgentRunSession `json:"sessions"`
		Total    int               `json:"total"`
	}
	if err := c.getJSON(path, &out); err != nil {
		return nil, 0, err
	}
	if out.Sessions == nil {
		out.Sessions = []AgentRunSession{}
	}
	if out.Total < len(out.Sessions) {
		out.Total = len(out.Sessions)
	}
	return out.Sessions, out.Total, nil
}

// DialAgentRunAttach upgrades GET /api/agent-run/sessions/{id}/attach to a
// WebSocket with bearer auth. Caller owns the connection and must Close it.
func (c *Client) DialAgentRunAttach(sessionID string) (*websocket.Conn, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id required")
	}
	wsURL, err := c.agentRunAttachWSURL(sessionID)
	if err != nil {
		return nil, err
	}
	hdr := http.Header{}
	if c.Token != "" {
		hdr.Set("Authorization", "Bearer "+c.Token)
	}
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	conn, resp, err := dialer.Dial(wsURL, hdr)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil, readAPIError(resp)
			}
		}
		return nil, fmt.Errorf("agent-run attach dial: %w", err)
	}
	return conn, nil
}

// AttachAgentRunSession dials attach and bridges binary frames to stdin/stdout
// until the remote side closes or an IO error occurs.
// onRestore is optional and runs on exit (local TTY restore hook).
func (c *Client) AttachAgentRunSession(sessionID string, stdin io.Reader, stdout io.Writer, onRestore func()) error {
	conn, err := c.DialAgentRunAttach(sessionID)
	if err != nil {
		return err
	}
	defer conn.Close()
	if onRestore != nil {
		defer onRestore()
	}

	errCh := make(chan error, 2)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, readErr := stdin.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					errCh <- werr
					return
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					errCh <- nil
					return
				}
				errCh <- readErr
				return
			}
		}
	}()
	go func() {
		for {
			mt, data, rerr := conn.ReadMessage()
			if rerr != nil {
				errCh <- rerr
				return
			}
			if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
				if _, werr := stdout.Write(data); werr != nil {
					errCh <- werr
					return
				}
			}
		}
	}()

	err = <-errCh
	if err == nil || err == io.EOF {
		return nil
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return nil
	}
	return err
}

// AgentRunStatusReport is multi-layer status from GET /api/agent-run/sessions/{id}/status.
type AgentRunStatusReport struct {
	Session   string                    `json:"session"`
	Status    string                    `json:"status"`
	Workspace string                    `json:"workspace,omitempty"`
	Process   AgentRunProcessLayer      `json:"process"`
	Terminal  AgentRunTerminalLayer     `json:"terminal"`
	Runner    AgentRunRunnerLayer       `json:"runner"`
	Resume    AgentRunResumeLayer       `json:"resume"`
}

// AgentRunProcessLayer is process status.
type AgentRunProcessLayer struct {
	Status string `json:"status"`
	PID    int    `json:"pid,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

// AgentRunTerminalLayer is terminal status.
type AgentRunTerminalLayer struct {
	Status   string `json:"status"`
	ID       string `json:"id,omitempty"`
	Listen   string `json:"listen,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Sendable string `json:"sendable,omitempty"`
}

// AgentRunRunnerLayer is runner bind status.
type AgentRunRunnerLayer struct {
	Status    string `json:"status"`
	Kind      string `json:"kind,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Exited    *bool  `json:"exited"`
}

// AgentRunResumeLayer is resume readiness.
type AgentRunResumeLayer struct {
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

// AgentRunResumeOpts is the body for POST resume.
type AgentRunResumeOpts struct {
	Open   bool   `json:"open"`
	Prompt string `json:"prompt,omitempty"`
}

// AgentRunRunOpts is the body for POST /api/agent-run/run.
type AgentRunRunOpts struct {
	SessionID        string `json:"session_id,omitempty"`
	Prompt           string `json:"prompt,omitempty"`
	Dir              string `json:"dir,omitempty"`
	Open             bool   `json:"open,omitempty"`
	Detach           bool   `json:"detach,omitempty"`
	JSON             bool   `json:"json,omitempty"`
	AutoSendOrResume bool   `json:"auto_send_or_resume,omitempty"`
	AgentRunner      string `json:"agent_runner,omitempty"`
	Model            string `json:"model,omitempty"`
}

// AgentRunRunResult is returned from a successful run.
type AgentRunRunResult struct {
	SessionID  string `json:"session_id"`
	TerminalID string `json:"terminal_id,omitempty"`
}

// RunAgentRunSession POSTs /api/agent-run/run.
func (c *Client) RunAgentRunSession(opts AgentRunRunOpts) (*AgentRunRunResult, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/agent-run/run", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out AgentRunRunResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetAgentRunHome calls GET /api/agent-run/status and returns the store home path.
func (c *Client) GetAgentRunHome() (string, error) {
	var out struct {
		Home string `json:"home"`
	}
	if err := c.getJSON("/api/agent-run/status", &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Home) == "" {
		return "", fmt.Errorf("server returned empty agent-run home")
	}
	return out.Home, nil
}

// StatusAgentRunSession calls GET /api/agent-run/sessions/{id}/status.
func (c *Client) StatusAgentRunSession(sessionID string) (*AgentRunStatusReport, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id required")
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/status"
	var out AgentRunStatusReport
	if err := c.getJSON(path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AgentRunSendOpts is the body for POST send.
type AgentRunSendOpts struct {
	Message  string `json:"message"`
	NoWait   bool   `json:"no_wait,omitempty"`
	NoSubmit bool   `json:"no_submit,omitempty"`
}

// SendAgentRunMessage POSTs a follow-up message; returns msg_id.
func (c *Client) SendAgentRunMessage(sessionID, message string, opts AgentRunSendOpts) (msgID string, err error) {
	sessionID = strings.TrimSpace(sessionID)
	message = strings.TrimSpace(message)
	if sessionID == "" || message == "" {
		return "", fmt.Errorf("send requires session-id and message")
	}
	opts.Message = message
	body, err := json.Marshal(opts)
	if err != nil {
		return "", err
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/send"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", readAPIError(resp)
	}
	var out struct {
		MsgID string `json:"msg_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.MsgID, nil
}

// AgentRunMsgStatus GETs message status (pending|delivered).
func (c *Client) AgentRunMsgStatus(sessionID, msgID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	msgID = strings.TrimSpace(msgID)
	if sessionID == "" || msgID == "" {
		return "", fmt.Errorf("msg status requires session-id/message-id")
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/messages/" + url.PathEscape(msgID)
	var out struct {
		Status string `json:"status"`
	}
	if err := c.getJSON(path, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

// AgentRunMsgCancel DELETEs (cancels) a queued message.
func (c *Client) AgentRunMsgCancel(sessionID, msgID string) error {
	sessionID = strings.TrimSpace(sessionID)
	msgID = strings.TrimSpace(msgID)
	if sessionID == "" || msgID == "" {
		return fmt.Errorf("msg cancel requires session-id/message-id")
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/messages/" + url.PathEscape(msgID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// AgentRunSnapshot GETs sanitized TTY text.
func (c *Client) AgentRunSnapshot(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("snapshot requires session-id")
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/snapshot"
	var out struct {
		Text string `json:"text"`
	}
	if err := c.getJSON(path, &out); err != nil {
		return "", err
	}
	return out.Text, nil
}

// AgentRunWatch streams watch output to w until the server finishes.
func (c *Client) AgentRunWatch(sessionID string, w io.Writer) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("watch requires session-id")
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/watch"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// AgentRunKill POSTs kill; dryRun reports without terminating.
func (c *Client) AgentRunKill(sessionID string, dryRun bool) (report string, err error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("kill requires session-id")
	}
	body, err := json.Marshal(map[string]bool{"dry_run": dryRun})
	if err != nil {
		return "", err
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/kill"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", readAPIError(resp)
	}
	var out struct {
		Report string `json:"report"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Report, nil
}

// ResumeAgentRunSession calls POST /api/agent-run/sessions/{id}/resume.
func (c *Client) ResumeAgentRunSession(sessionID string, opts AgentRunResumeOpts) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id required")
	}
	body, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	path := "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/resume"
	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) agentRunAttachWSURL(sessionID string) (string, error) {
	base := strings.TrimRight(c.Server, "/")
	if base == "" {
		return "", fmt.Errorf("server URL is empty")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "ws", "wss":
		// ok
	default:
		return "", fmt.Errorf("unsupported server scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/agent-run/sessions/" + url.PathEscape(sessionID) + "/attach"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
