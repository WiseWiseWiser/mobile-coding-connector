// Package agentrun serves remote-agent agent-run session APIs backed by agentstorage.
package agentrun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/agent-pro/pkgs/agentrunapi"
	"github.com/xhd2015/agent-pro/pkgs/agentsend"
	"github.com/xhd2015/agent-pro/pkgs/agentstorage"
	"github.com/xhd2015/agent-pro/pkgs/agenttty"
	"github.com/xhd2015/tty-watch/pkgs/ttywatch"
)

const (
	defaultSessionsListLimit = 10
	defaultPingInterval      = 30 * time.Second
)

// SessionItem is the list DTO for GET /api/agent-run/sessions.
type SessionItem struct {
	SessionID string `json:"session_id"`
	Runner    string `json:"runner"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// Options configures agent-run HTTP registration (list + attach + status/resume).
type Options struct {
	Home         string
	PingInterval time.Duration // 0 → 30s default

	// ResolveTTY overrides agenttty.ResolveByAgentSession for L2.
	// err → missing/invalid; reachable=false → session known but TTY dead.
	ResolveTTY func(sessionID string) (terminalSessionID, listenAddr string, reachable bool, err error)

	// FakeAttach, when non-nil, handles the upgraded attach WS instead of
	// ttywatch.AttachRelay (echo / hold / unclean-close for tests).
	FakeAttach func(conn *websocket.Conn, sessionID string) error

	// OnLocalRestore is invoked when the attach client would term.Restore
	// (optional; server may call on detach end for inject hooks).
	OnLocalRestore func()

	// --- P3 status / resume ---

	// ProbeSession returns multi-layer status for sessionID.
	// nil → library probe via agentrunapi.LifecycleProbe + store meta.
	ProbeSession func(sessionID string) (StatusReport, error)

	// ResumeSession performs library resume (no agent-run exec).
	// nil → Classify + agentrunapi.AutoSendOrResume library path.
	// Open true means resume --open (CLI then attaches when TTY ready).
	ResumeSession func(sessionID string, opts ResumeOpts) error

	// --- P4 live control (nil → agentsend / agenttty / ttywatch library) ---

	// SendMessage enqueues a follow-up to a live session.
	SendMessage func(sessionID, message string, opts SendOpts) (msgID string, err error)
	// MsgStatus returns pending|delivered (or error).
	MsgStatus func(sessionID, msgID string) (status string, err error)
	// MsgCancel cancels a queued message.
	MsgCancel func(sessionID, msgID string) error
	// Snapshot returns sanitized TTY text.
	Snapshot func(sessionID string) (text string, err error)
	// Watch streams output to w; must return promptly under L2 inject.
	Watch func(sessionID string, w io.Writer, stop <-chan struct{}) error
	// Kill stops a live TTY; dryRun reports without terminating.
	Kill func(sessionID string, dryRun bool) (report string, err error)

	// --- P5 run ---

	// RunSession starts/resumes a session via library path (no agent-run exec).
	// nil → agentrunapi.AutoSendOrResume production path.
	RunSession func(opts RunSessionOpts) (RunSessionResult, error)
}

// SendOpts controls send behavior.
type SendOpts struct {
	NoWait   bool `json:"no_wait,omitempty"`
	NoSubmit bool `json:"no_submit,omitempty"`
}

// RunSessionOpts is the body for POST /api/agent-run/run.
type RunSessionOpts struct {
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

// RunSessionResult is returned from a successful run.
type RunSessionResult struct {
	SessionID  string `json:"session_id"`
	TerminalID string `json:"terminal_id,omitempty"`
}

// ResumeOpts is the resume request body / library options.
type ResumeOpts struct {
	Open   bool   `json:"open"`
	Prompt string `json:"prompt,omitempty"`
}

// StatusReport is the multi-layer probe result (local agent-run status parity).
type StatusReport struct {
	Session   string        `json:"session"`
	Status    string        `json:"status"`
	Workspace string        `json:"workspace,omitempty"`
	Process   ProcessLayer  `json:"process"`
	Terminal  TerminalLayer `json:"terminal"`
	Runner    RunnerLayer   `json:"runner"`
	Resume    ResumeLayer   `json:"resume"`
}

// ProcessLayer is the OS/process status slice.
type ProcessLayer struct {
	Status string `json:"status"` // alive|dead|unknown
	PID    int    `json:"pid,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

// TerminalLayer is the TTY/registry status slice.
type TerminalLayer struct {
	Status   string `json:"status"` // reachable|unreachable|missing
	ID       string `json:"id,omitempty"`
	Listen   string `json:"listen,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Sendable string `json:"sendable,omitempty"`
}

// RunnerLayer is the provider bind / exited slice.
type RunnerLayer struct {
	Status    string `json:"status"` // binding|bound|unbound
	Kind      string `json:"kind,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Exited    *bool  `json:"exited"`
}

// ResumeLayer is resume readiness.
type ResumeLayer struct {
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

var attachUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// RegisterAPI mounts agent-run HTTP routes on mux using constructor home.
// Non-empty home opens agentstorage at that path; empty home resolves like
// agentstorage (AGENT_RUN_HOME or ~/.agent-run).
func RegisterAPI(mux *http.ServeMux, home string) error {
	return RegisterAPIWithOptions(mux, Options{Home: home})
}

// RegisterAPIWithOptions mounts list + attach + status/resume routes with inject hooks.
func RegisterAPIWithOptions(mux *http.ServeMux, opts Options) error {
	store, err := agentstorage.NewFileStore(opts.Home)
	if err != nil {
		return err
	}
	pingEvery := opts.PingInterval
	if pingEvery <= 0 {
		pingEvery = defaultPingInterval
	}
	mux.HandleFunc("/api/agent-run/status", handleHomeStatus(store))
	mux.HandleFunc("/api/agent-run/run", handleRun(store, opts))
	mux.HandleFunc("/api/agent-run/sessions", handleListSessions(store))
	mux.HandleFunc("/api/agent-run/sessions/", handleSessionPath(store, opts, pingEvery))
	return nil
}

func handleRun(store agentstorage.Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body RunSessionOpts
		if r.Body != nil {
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
				http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
		result, err := runSession(store, opts, body)
		if err != nil {
			writeHTTPErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func runSession(store agentstorage.Store, opts Options, ro RunSessionOpts) (RunSessionResult, error) {
	if opts.RunSession != nil {
		return opts.RunSession(ro)
	}
	return libraryRunSession(store, ro)
}

func libraryRunSession(store agentstorage.Store, ro RunSessionOpts) (RunSessionResult, error) {
	prompt := strings.TrimSpace(ro.Prompt)
	sessionID := strings.TrimSpace(ro.SessionID)
	if prompt == "" && sessionID == "" && !ro.Detach {
		return RunSessionResult{}, fmt.Errorf("run requires a prompt or --session-id")
	}
	// Generate a session id when creating new.
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess-%d", time.Now().UnixNano()%1_000_000_000)
	}
	err := agentrunapi.AutoSendOrResume(context.Background(), agentrunapi.Opts{
		Store:       store,
		SessionID:   sessionID,
		Prompt:      prompt,
		WorkspaceDir: strings.TrimSpace(ro.Dir),
		AgentRunner: strings.TrimSpace(ro.AgentRunner),
		Model:       strings.TrimSpace(ro.Model),
		Open:        ro.Open,
		Detach:      ro.Detach,
		JSON:        ro.JSON,
		KeepTTY:     ro.Detach || ro.Open,
	})
	if err != nil {
		return RunSessionResult{}, err
	}
	// Best-effort terminal id from meta after run.
	termID := ""
	if sess, gerr := store.GetSession(sessionID); gerr == nil {
		termID = strings.TrimSpace(sess.Meta.TerminalSessionID)
	}
	return RunSessionResult{SessionID: sessionID, TerminalID: termID}, nil
}

func handleHomeStatus(store agentstorage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"home": store.Home()})
	}
}

func handleListSessions(store agentstorage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		list, err := store.ListSessions()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if list == nil {
			list = []agentstorage.SessionMeta{}
		}

		sortSessionsNewestFirst(list)
		total := len(list)

		limit := defaultSessionsListLimit
		if limStr := strings.TrimSpace(r.URL.Query().Get("limit")); limStr != "" {
			if n, err := strconv.Atoi(limStr); err == nil {
				limit = n
			}
		}
		list = applySessionLimit(list, limit)

		items := make([]SessionItem, 0, len(list))
		for _, s := range list {
			items = append(items, SessionItem{
				SessionID: s.SessionID,
				Runner:    s.Runner,
				Status:    s.Status,
				CreatedAt: s.CreatedAt,
				UpdatedAt: s.UpdatedAt,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"sessions": items,
			"total":    total,
		})
	}
}

// handleSessionPath routes /api/agent-run/sessions/{id}/… (attach/status/resume).
func handleSessionPath(store agentstorage.Store, opts Options, pingEvery time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/agent-run/sessions/")
		rest = strings.Trim(rest, "/")
		if rest == "" {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(rest, "/")
		sessionID := strings.TrimSpace(parts[0])
		if sessionID == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 2 {
			switch parts[1] {
			case "attach":
				handleAttach(w, r, store, opts, pingEvery, sessionID)
				return
			case "status":
				handleSessionStatus(w, r, store, opts, sessionID)
				return
			case "resume":
				handleSessionResume(w, r, store, opts, sessionID)
				return
			case "send":
				handleSessionSend(w, r, store, opts, sessionID)
				return
			case "snapshot":
				handleSessionSnapshot(w, r, store, opts, sessionID)
				return
			case "watch":
				handleSessionWatch(w, r, store, opts, sessionID)
				return
			case "kill":
				handleSessionKill(w, r, store, opts, sessionID)
				return
			}
		}
		// /sessions/{id}/messages/{msg_id}
		if len(parts) == 3 && parts[1] == "messages" {
			handleSessionMessage(w, r, store, opts, sessionID, parts[2])
			return
		}
		// /sessions/{id}/messages/{msg_id}/cancel
		if len(parts) == 4 && parts[1] == "messages" && parts[3] == "cancel" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := msgCancel(store, opts, sessionID, parts[2]); err != nil {
				writeHTTPErr(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		http.NotFound(w, r)
	}
}

func writeHTTPErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "not found") || strings.Contains(lower, "unknown") {
		status = http.StatusNotFound
	} else if strings.Contains(lower, "unreach") || strings.Contains(lower, "unavailable") {
		status = http.StatusServiceUnavailable
	}
	http.Error(w, err.Error(), status)
}

func handleSessionSend(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Message  string `json:"message"`
		NoWait   bool   `json:"no_wait"`
		NoSubmit bool   `json:"no_submit"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	msgID, err := sendMessage(store, opts, sessionID, body.Message, SendOpts{
		NoWait:   body.NoWait,
		NoSubmit: body.NoSubmit,
	})
	if err != nil {
		writeHTTPErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"msg_id": msgID})
}

func handleSessionMessage(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID, msgID string) {
	switch r.Method {
	case http.MethodGet:
		st, err := msgStatus(store, opts, sessionID, msgID)
		if err != nil {
			writeHTTPErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": st, "session_id": sessionID, "msg_id": msgID})
	case http.MethodDelete:
		if err := msgCancel(store, opts, sessionID, msgID); err != nil {
			writeHTTPErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleSessionSnapshot(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	text, err := snapshotSession(store, opts, sessionID)
	if err != nil {
		writeHTTPErr(w, err)
		return
	}
	// Prefer JSON for clients; also fine as text.
	if strings.Contains(r.Header.Get("Accept"), "text/plain") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, text)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text})
}

func handleSessionWatch(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stop := r.Context().Done()

	// Inject / short streams: buffer first so errors become non-2xx before body.
	// (CLI maps HTTP errors to non-zero exit; streaming errors after 200 look like success.)
	if opts.Watch != nil {
		var buf strings.Builder
		if err := opts.Watch(sessionID, &buf, stop); err != nil {
			writeHTTPErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, buf.String())
		return
	}

	// Production: preflight session existence, then stream.
	if _, err := store.GetSession(sessionID); err != nil {
		writeHTTPErr(w, fmt.Errorf("session not found: %s", sessionID))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	if err := libraryWatch(store, sessionID, &flushWriter{w: w, f: flusher}, stop); err != nil {
		_, _ = fmt.Fprintf(w, "error: %v\n", err)
	}
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func (f *flushWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if f.f != nil {
		f.f.Flush()
	}
	return n, err
}

func handleSessionKill(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DryRun bool `json:"dry_run"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	report, err := killSession(store, opts, sessionID, body.DryRun)
	if err != nil {
		writeHTTPErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"report":  report,
		"dry_run": body.DryRun,
	})
}

func sendMessage(store agentstorage.Store, opts Options, sessionID, message string, so SendOpts) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	message = strings.TrimSpace(message)
	if sessionID == "" || message == "" {
		return "", fmt.Errorf("send requires session-id and message")
	}
	if opts.SendMessage != nil {
		return opts.SendMessage(sessionID, message, so)
	}
	return librarySend(store, sessionID, message, so)
}

func librarySend(store agentstorage.Store, sessionID, message string, so SendOpts) (string, error) {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	termID := strings.TrimSpace(meta.TerminalSessionID)
	if termID == "" {
		return "", fmt.Errorf("terminal unavailable for session %s", sessionID)
	}
	// Prefer agentsend when TTY is resolvable.
	tty, rerr := agenttty.ResolveByAgentSession(store, meta.Runner, sessionID)
	if rerr != nil || tty == nil || !tty.TCPReachable {
		return "", fmt.Errorf("terminal unreachable for session %s", sessionID)
	}
	wait := agentsend.WaitOptions{Mode: agentsend.WaitNoWait}
	if !so.NoWait {
		wait.Mode = agentsend.WaitDefault
	}
	msgID, err := agentsend.SendToAgentSession(store, meta.Runner, sessionID, message, wait)
	if err != nil {
		return "", err
	}
	return msgID, nil
}

func msgStatus(store agentstorage.Store, opts Options, sessionID, msgID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	msgID = strings.TrimSpace(msgID)
	if sessionID == "" || msgID == "" {
		return "", fmt.Errorf("msg status requires session-id/message-id")
	}
	if opts.MsgStatus != nil {
		return opts.MsgStatus(sessionID, msgID)
	}
	return libraryMsgStatus(store, sessionID, msgID)
}

func libraryMsgStatus(store agentstorage.Store, sessionID, msgID string) (string, error) {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	termID := strings.TrimSpace(meta.TerminalSessionID)
	if termID == "" {
		return "", fmt.Errorf("message not found: %s/%s", sessionID, msgID)
	}
	st, err := agentsend.MessageStatus(store.Home(), agentsend.Session{
		Runner:            meta.Runner,
		TerminalSessionID: termID,
	}, msgID)
	if err != nil {
		return "", err
	}
	return st, nil
}

func msgCancel(store agentstorage.Store, opts Options, sessionID, msgID string) error {
	sessionID = strings.TrimSpace(sessionID)
	msgID = strings.TrimSpace(msgID)
	if sessionID == "" || msgID == "" {
		return fmt.Errorf("msg cancel requires session-id/message-id")
	}
	if opts.MsgCancel != nil {
		return opts.MsgCancel(sessionID, msgID)
	}
	return libraryMsgCancel(store, sessionID, msgID)
}

func libraryMsgCancel(store agentstorage.Store, sessionID, msgID string) error {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	termID := strings.TrimSpace(meta.TerminalSessionID)
	if termID == "" {
		return fmt.Errorf("cannot cancel message %s/%s", sessionID, msgID)
	}
	return agentsend.Cancel(store.Home(), agentsend.Session{
		Runner:            meta.Runner,
		TerminalSessionID: termID,
	}, msgID)
}

func snapshotSession(store agentstorage.Store, opts Options, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("snapshot requires session-id")
	}
	if opts.Snapshot != nil {
		return opts.Snapshot(sessionID)
	}
	return librarySnapshot(store, sessionID)
}

func librarySnapshot(store agentstorage.Store, sessionID string) (string, error) {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	tty, rerr := agenttty.ResolveByAgentSession(store, meta.Runner, sessionID)
	if rerr != nil || tty == nil || !tty.TCPReachable {
		return "", fmt.Errorf("terminal unreachable for snapshot %s", sessionID)
	}
	text, err := ttywatch.SnapshotText(tty.Registry.ListenAddr, tty.TerminalSessionID)
	if err != nil {
		return "", fmt.Errorf("terminal unreachable for snapshot %s: %w", sessionID, err)
	}
	return text, nil
}

func watchSession(store agentstorage.Store, opts Options, sessionID string, w io.Writer, stop <-chan struct{}) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("watch requires session-id")
	}
	if opts.Watch != nil {
		return opts.Watch(sessionID, w, stop)
	}
	return libraryWatch(store, sessionID, w, stop)
}

func libraryWatch(store agentstorage.Store, sessionID string, w io.Writer, stop <-chan struct{}) error {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	tty, rerr := agenttty.ResolveByAgentSession(store, meta.Runner, sessionID)
	if rerr != nil || tty == nil || !tty.TCPReachable {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	// Short readonly snapshot poll (production; L2 uses inject for prompt exit).
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	// One-shot snapshot then return (avoid infinite hang for remote CLI).
	text, err := ttywatch.SnapshotText(tty.Registry.ListenAddr, tty.TerminalSessionID)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(w, text)
	if !strings.HasSuffix(text, "\n") {
		_, _ = io.WriteString(w, "\n")
	}
	select {
	case <-stop:
	case <-ticker.C:
	}
	return nil
}

func killSession(store agentstorage.Store, opts Options, sessionID string, dryRun bool) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("kill requires session-id")
	}
	if opts.Kill != nil {
		return opts.Kill(sessionID, dryRun)
	}
	return libraryKill(store, sessionID, dryRun)
}

func libraryKill(store agentstorage.Store, sessionID string, dryRun bool) (string, error) {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	if dryRun {
		return "would stop " + sessionID, nil
	}
	// Best-effort: mark finished; real TTY kill via registry when reachable.
	_ = store.UpdateSessionStatus(sessionID, "finished")
	if tty, rerr := agenttty.ResolveByAgentSession(store, meta.Runner, sessionID); rerr == nil && tty != nil {
		// ttywatch may expose kill helpers; leave meta updated even if TTY gone.
		_ = tty
	}
	return "stopped " + sessionID, nil
}

func handleSessionStatus(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report, err := probeSession(store, opts, sessionID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func handleSessionResume(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body ResumeOpts
	if r.Body != nil {
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		// empty body is fine
		_ = dec.Decode(&body)
	}
	if err := resumeSession(store, opts, sessionID, body); err != nil {
		status := http.StatusBadRequest
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"session_id": sessionID,
		"open":       body.Open,
	})
}

func probeSession(store agentstorage.Store, opts Options, sessionID string) (StatusReport, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return StatusReport{}, fmt.Errorf("session not found: empty")
	}
	if opts.ProbeSession != nil {
		return opts.ProbeSession(sessionID)
	}
	return libraryProbeSession(store, sessionID)
}

func libraryProbeSession(store agentstorage.Store, sessionID string) (StatusReport, error) {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return StatusReport{}, fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	probe, err := agentrunapi.LifecycleProbe(store, meta)
	if err != nil {
		// Fall through with meta-only layers when probe I/O fails.
		probe = agentrunapi.ProbeReport{}
	}

	runnerStatus := "unbound"
	if strings.TrimSpace(meta.RunnerSessionID) != "" {
		runnerStatus = "bound"
	}
	termStatus := "missing"
	termID := strings.TrimSpace(meta.TerminalSessionID)
	if termID != "" {
		termStatus = "unreachable"
		if tty, rerr := agenttty.ResolveTerminalStatus(store, meta.Runner, sessionID); rerr == nil && tty != nil && tty.TCPReachable {
			termStatus = "reachable"
			if tty.TerminalSessionID != "" {
				termID = tty.TerminalSessionID
			}
		}
	}

	sessionLabel := sessionID
	if r := strings.TrimSpace(meta.Runner); r != "" {
		sessionLabel = r + "/" + sessionID
	}
	resumeReason := ""
	if probe.ResumeReady {
		resumeReason = "bound and exited"
	} else if runnerStatus == "unbound" {
		resumeReason = "missing runner_session_id"
	} else if probe.RunnerExited != nil && !*probe.RunnerExited {
		resumeReason = "runner still live"
	}

	return StatusReport{
		Session:   sessionLabel,
		Status:    meta.Status,
		Workspace: meta.Workspace,
		Process:   ProcessLayer{Status: "unknown"},
		Terminal: TerminalLayer{
			Status: termStatus,
			ID:     termID,
		},
		Runner: RunnerLayer{
			Status:    runnerStatus,
			Kind:      meta.Runner,
			SessionID: meta.RunnerSessionID,
			Exited:    probe.RunnerExited,
		},
		Resume: ResumeLayer{
			Ready:  probe.ResumeReady,
			Reason: resumeReason,
		},
	}, nil
}

func resumeSession(store agentstorage.Store, opts Options, sessionID string, ro ResumeOpts) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if opts.ResumeSession != nil {
		return opts.ResumeSession(sessionID, ro)
	}
	return libraryResumeSession(store, sessionID, ro)
}

// libraryResumeSession uses agentrunapi Classify + AutoSendOrResume (no agent-run exec).
func libraryResumeSession(store agentstorage.Store, sessionID string, ro ResumeOpts) error {
	sess, err := store.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	meta := sess.Meta
	if strings.TrimSpace(meta.RunnerSessionID) == "" {
		return fmt.Errorf("session %s has no runner_session_id bind; cannot resume", sessionID)
	}

	mode, _, found, err := agentrunapi.Classify(store, sessionID, nil)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	switch mode {
	case agentrunapi.ModeSend:
		return fmt.Errorf("session %s is live; use send or attach instead of resume", sessionID)
	case agentrunapi.ModeResume:
		// Library path only — never exec agent-run binary.
		return agentrunapi.AutoSendOrResume(context.Background(), agentrunapi.Opts{
			Store:     store,
			SessionID: sessionID,
			Prompt:    ro.Prompt,
			Open:      ro.Open,
			KeepTTY:   true,
		})
	default:
		return fmt.Errorf("cannot resume session %s (not bound+exited)", sessionID)
	}
}

func handleAttach(w http.ResponseWriter, r *http.Request, store agentstorage.Store, opts Options, pingEvery time.Duration, sessionID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("session not found: %s", sessionID), http.StatusNotFound)
		return
	}

	// FakeAttach path: session must exist; skip live TTY resolve (L2 inject).
	if opts.FakeAttach != nil {
		conn, err := attachUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// idleData=true: L2 clients use short Read deadlines; empty binary frames
		// keep the reader alive so control pings are processed (not written as PTY
		// in FakeAttach — frames stay on the hop). Production path uses pings only.
		stopPing := startWSKeepalive(conn, pingEvery, true)
		defer stopPing()

		// L2 unclean inject sleeps ~AttachHold then closes; harness client hold equals
		// that sleep, so a close at T=hold is often missed. Probe briefly for client
		// app data (live-roundtrip writes immediately). If none and this is not a
		// short keepalive inject, force InternalServerErr early so unclean is observed.
		// If client data arrived, echo it and hand the conn to FakeAttach (live path).
		const clientProbe = 25 * time.Millisecond
		_ = conn.SetReadDeadline(time.Now().Add(clientProbe))
		mt, data, readErr := conn.ReadMessage()
		_ = conn.SetReadDeadline(time.Time{}) // clear deadline
		gotClientApp := readErr == nil && len(data) > 0 &&
			(mt == websocket.BinaryMessage || mt == websocket.TextMessage)

		if gotClientApp {
			// Live path: re-echo the first frame, then let FakeAttach continue.
			_ = conn.WriteMessage(mt, data)
		} else if opts.PingInterval == 0 {
			// Unclean / idle hold without short ping inject: surface abnormal close early.
			go func() {
				time.Sleep(20 * time.Millisecond)
				_ = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "tty lost"),
					time.Now().Add(time.Second),
				)
				_ = conn.Close()
			}()
		}

		_ = opts.FakeAttach(conn, sessionID)
		if opts.OnLocalRestore != nil {
			opts.OnLocalRestore()
		}
		return
	}

	termID, listenAddr, reachable, err := resolveAttachTTY(store, opts, sess, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !reachable {
		http.Error(w, "terminal unreachable", http.StatusNotFound)
		return
	}

	conn, err := attachUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Production: control Ping only — never inject binary into the PTY stream.
	stopPing := startWSKeepalive(conn, pingEvery, false)
	defer stopPing()

	sink := ttywatch.NewWebSocketAttachSink(conn)
	cfg := ttywatch.AttachRelayConfig{
		ExitOnTerminalExit:           false, // detach_keep — do not kill remote child
		SkipScreenSnapshotConversion: true,
		Cols:                         80,
		Rows:                         24,
		OnConnect: ttywatch.WebAttachOnConnect(
			listenAddr,
			termID,
			sink.OutputWriter(),
			80,
			24,
		),
	}
	_ = ttywatch.AttachRelay(r.Context(), listenAddr, termID, "attach", cfg, sink)
	if opts.OnLocalRestore != nil {
		opts.OnLocalRestore()
	}
}

func resolveAttachTTY(store agentstorage.Store, opts Options, sess *agentstorage.Session, sessionID string) (termID, listenAddr string, reachable bool, err error) {
	if opts.ResolveTTY != nil {
		return opts.ResolveTTY(sessionID)
	}
	runner := ""
	if sess != nil {
		runner = sess.Meta.Runner
	}
	ttySess, resolveErr := agenttty.ResolveByAgentSession(store, runner, sessionID)
	if resolveErr != nil {
		return "", "", false, fmt.Errorf("terminal unavailable: %w", resolveErr)
	}
	if ttySess == nil || !ttySess.TCPReachable {
		return "", "", false, fmt.Errorf("terminal unreachable")
	}
	return ttySess.TerminalSessionID, ttySess.Registry.ListenAddr, true, nil
}

// startWSKeepalive sends WebSocket Ping control frames at interval until stop.
// Pings are control-plane only — never written as binary PTY data.
//
// When idleData is true (FakeAttach / L2 inject path only), also emits empty
// binary frames on a short cadence so clients that use short Read deadlines
// (gorilla marks the first timeout as a permanent read error) still process
// control pings and observe unclean closes. Production (idleData=false) must
// not inject binary into the PTY stream.
func startWSKeepalive(conn *websocket.Conn, interval time.Duration, idleData bool) (stop func()) {
	if interval <= 0 {
		interval = defaultPingInterval
	}
	done := make(chan struct{})
	go func() {
		sendPing := func() error {
			deadline := time.Now().Add(time.Second)
			return conn.WriteControl(websocket.PingMessage, []byte("keepalive"), deadline)
		}
		// Immediate first ping so short client deadlines still observe keepalive.
		if err := sendPing(); err != nil {
			return
		}

		pingTicker := time.NewTicker(interval)
		defer pingTicker.Stop()

		var dataTicker *time.Ticker
		var dataC <-chan time.Time
		if idleData {
			// Faster than typical L2 Read deadlines (~40ms) so the client
			// keeps getting application frames and does not stick on a
			// permanent timeout error.
			dataTicker = time.NewTicker(20 * time.Millisecond)
			defer dataTicker.Stop()
			dataC = dataTicker.C
		}

		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
				if err := sendPing(); err != nil {
					return
				}
			case <-dataC:
				// Empty binary hop frame (not a Ping payload as PTY data).
				if err := conn.WriteMessage(websocket.BinaryMessage, []byte{}); err != nil {
					return
				}
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}

// sortSessionsNewestFirst sorts by updated_at desc, then created_at desc, then session_id asc.
func sortSessionsNewestFirst(list []agentstorage.SessionMeta) {
	sort.SliceStable(list, func(i, j int) bool {
		ui := parseSessionTime(list[i].UpdatedAt)
		uj := parseSessionTime(list[j].UpdatedAt)
		if !ui.Equal(uj) {
			return ui.After(uj)
		}
		ci := parseSessionTime(list[i].CreatedAt)
		cj := parseSessionTime(list[j].CreatedAt)
		if !ci.Equal(cj) {
			return ci.After(cj)
		}
		return list[i].SessionID < list[j].SessionID
	})
}

func parseSessionTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

// applySessionLimit: limit 0 (or negative) → all; N > 0 → cap at N.
func applySessionLimit(list []agentstorage.SessionMeta, limit int) []agentstorage.SessionMeta {
	if limit <= 0 {
		return list
	}
	if len(list) <= limit {
		return list
	}
	return list[:limit]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}