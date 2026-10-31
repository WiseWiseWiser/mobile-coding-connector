package browser

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// RegisterAPI mounts the browser endpoints. They sit behind the server's auth
// middleware like every other /api route, so no extra token check is needed.
func RegisterAPI(mux *http.ServeMux) {
	m := Default()
	mux.HandleFunc("/api/browser/status", m.handleStatus)
	mux.HandleFunc("/api/browser/ensure", m.handleEnsure)
	mux.HandleFunc("/api/browser/navigate", m.handleNavigate)
	mux.HandleFunc("/api/browser/preset", m.handlePreset)
	mux.HandleFunc("/api/browser/mode", m.handleMode)
	mux.HandleFunc("/api/browser/refresh", m.handleRefresh)
	mux.HandleFunc("/api/browser/sessions", m.handleSessions)
	mux.HandleFunc("/api/browser/session", m.handleSession)
	mux.HandleFunc("/api/browser/window/attach", m.handleWindowAttach)
	mux.HandleFunc("/api/browser/window/close", m.handleWindowClose)
	mux.HandleFunc("/api/browser/tab", m.handleTab)
	mux.HandleFunc("/api/browser/frame", m.handleFrame)
	mux.HandleFunc("/api/browser/input", m.handleInput)
	mux.HandleFunc("/api/browser/ws", m.handleWS)
}

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

func (m *Manager) handleEnsure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := m.StartBrowser(ctx); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errBrowserStarting) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

func (m *Manager) handleNavigate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL       string `json:"url"`
		SessionID string `json:"session_id"`
		TabID     int64  `json:"tab_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	var err error
	if body.SessionID != "" || body.TabID != 0 {
		if body.SessionID == "" || body.TabID <= 0 {
			writeError(w, http.StatusBadRequest, "session_id and tab_id are both required")
			return
		}
		err = m.NavigateTab(ctx, body.SessionID, body.TabID, body.URL)
	} else {
		err = m.Navigate(ctx, body.URL)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

func (m *Manager) handlePreset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Preset string `json:"preset"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if err := m.SetPreset(ctx, body.Preset); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

func (m *Manager) handleMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if err := m.SetMode(body.Mode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

func (m *Manager) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	m.hub.kick()
	writeJSON(w, http.StatusOK, m.Snapshot())
}

// handleSessions lists the sessions whose Chrome windows are actually alive, the
// way the rail draws them: id, window, tabs, and for the current one its preset
// and capture stats.
func (m *Manager) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	resp := m.Sessions(ctx)
	if !resp.OK {
		writeJSON(w, http.StatusInternalServerError, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSession switches the current session. {"new":true} mints a fresh one and
// opens its page in a new Chrome window; {"session_id":"…"} switches to that id,
// reopening its window when it is gone.
func (m *Manager) handleSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
		New       bool   `json:"new"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	if !body.New && isSessionlessSessionID(strings.TrimSpace(body.SessionID)) {
		// Direct-capable daemon: a sessionless id names a plain window, so the
		// switch goes through the direct rail (no /go page). Otherwise keep the
		// dedicated attach endpoint, which registers a real session.
		if !m.DirectSupported() {
			writeError(w, http.StatusBadRequest, "discovered windows must be attached with /api/browser/window/attach")
			return
		}
	}
	if body.New {
		id, err := m.NewSession(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error(), "session_id": id})
			return
		}
		writeJSON(w, http.StatusOK, m.Snapshot())
		return
	}
	if err := m.SwitchSession(ctx, body.SessionID, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error(), "session_id": body.SessionID})
		return
	}
	writeJSON(w, http.StatusOK, m.Snapshot())
}

// handleWindowAttach adopts a window discovered by the "windows" job. The
// window id is the only selector: the server derives the reserved session id
// from it, opens /go in that exact window, and switches only after the
// extension confirms the registration belongs to the same window.
func (m *Manager) handleWindowAttach(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WindowID  int64  `json:"window_id"`
		SessionID string `json:"session_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	sessionID, err := m.AttachWindow(ctx, body.WindowID, body.SessionID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errInvalidWindowSession):
			status = http.StatusBadRequest
		case errors.Is(err, errAllWindowsUnsupported), errors.Is(err, errNoLiveWindow),
			errors.Is(err, errWindowGone), errors.Is(err, errWindowHasSession):
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"ok": false, "error": err.Error(), "window_id": body.WindowID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"session_id": sessionID,
		"window_id":  body.WindowID,
		"status":     m.Snapshot(),
	})
}

// handleWindowClose removes one whole Chrome window. Exactly one selector is
// accepted. Confirmation belongs to the caller: this endpoint is the action.
func (m *Manager) handleWindowClose(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WindowID  int64  `json:"window_id"`
		SessionID string `json:"session_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := m.CloseWindow(ctx, body.WindowID, body.SessionID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errWindowCloseTarget), errors.Is(err, errWindowCloseBoth),
			errors.Is(err, errInvalidWindowSession):
			status = http.StatusBadRequest
		case errors.Is(err, errWindowCloseUnsupported), errors.Is(err, errWindowCloseLast),
			errors.Is(err, errWindowCloseCurrent), errors.Is(err, errWindowGone),
			errors.Is(err, errWindowCloseNoAnchor):
			status = http.StatusConflict
		case errors.Is(err, errReadOnly):
			status = http.StatusFailedDependency
		case errors.Is(err, errNoLiveWindow):
			status = http.StatusConflict
		default:
			status = http.StatusBadGateway
		}
		resp := map[string]any{"ok": false, "error": err.Error(), "window_id": body.WindowID}
		if result != nil {
			resp["window_id"] = result.WindowID
			resp["closed_session_id"] = result.SessionID
		}
		writeJSON(w, status, resp)
		return
	}
	resp := map[string]any{
		"ok":               true,
		"closed_window_id": result.WindowID,
		"current":          m.sessionIDValue(),
	}
	if result.SessionID != "" {
		resp["closed_session_id"] = result.SessionID
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleTab activates, closes, or creates a tab. Creating takes a session id
// because the rail can add a tab to a session the user is not watching; closing
// the last tab is refused (409) before any job is sent.
func (m *Manager) handleTab(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID   string `json:"session_id"`
		Op          string `json:"op"`
		TabID       int64  `json:"tab_id"`
		URL         string `json:"url"`
		OperationID string `json:"operation_id"`
		Accept      *bool  `json:"accept"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	effective := body.SessionID
	if effective == "" {
		effective = m.sessionIDValue()
	}
	var info *sessionInfo
	var err error
	if body.Op == "confirm_close" {
		if body.OperationID == "" || body.Accept == nil {
			writeError(w, http.StatusBadRequest, "operation_id and explicit accept are required")
			return
		}
		info, err = m.ResolveTabClose(ctx, effective, body.OperationID, *body.Accept)
	} else {
		info, err = m.TabOp(ctx, body.SessionID, body.Op, body.TabID, body.URL)
	}
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errReadOnly):
			status = http.StatusFailedDependency
		}
		writeJSON(w, status, map[string]any{"ok": false, "error": err.Error(), "session_id": body.SessionID})
		return
	}
	resp := map[string]any{
		"ok":            true,
		"session_id":    effective,
		"tabs":          info.Tabs,
		"active_tab_id": info.Active,
	}
	if info.CloseConfirmation != nil {
		resp["close_confirmation"] = info.CloseConfirmation
	}
	if info.CloseOutcome != "" {
		resp["close_outcome"] = info.CloseOutcome
	}
	if effective == m.sessionIDValue() {
		resp["status"] = m.Snapshot()
	}
	writeJSON(w, http.StatusOK, resp)
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil && !errors.Is(err, http.ErrBodyReadAfterClose) {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": message})
}
