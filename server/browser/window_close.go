package browser

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// windowCloseFeature is the extension capability that permits removing a whole
// Chrome window. It is separate from "all-windows": an extension can list every
// window without being allowed to close one.
const windowCloseFeature = "window-close"

// windowCloseMinVersion is only an anchor preference; the feature list in the
// extension's answer is the real gate.
const windowCloseMinVersion = "1.0.22"

var (
	errWindowCloseUnsupported = errors.New("the browser-agent extension does not support closing windows (1.0.22+ required); reload it from the extension directory")
	errWindowCloseLast        = errors.New("cannot close the last Chrome window")
	errWindowCloseCurrent     = errors.New("cannot close the current window: open or attach another browser-agent window first")
	errWindowCloseTarget      = errors.New("provide either window_id or session_id")
	errWindowCloseBoth        = errors.New("provide window_id or session_id, not both")
	errWindowCloseNoAnchor    = errors.New("no other live browser-agent window can close that Chrome window")
)

// CloseWindowResult describes the window that was removed and the session the
// manager drives afterwards. Current is empty when the caller's current window
// was not the target.
type CloseWindowResult struct {
	WindowID  int64  `json:"window_id"`
	SessionID string `json:"session_id,omitempty"`
	Current   string `json:"current,omitempty"`
}

func (p *windowsPayload) supportsWindowClose() bool {
	return featureEnabled(p.Features, windowCloseFeature)
}

func extensionSupportsWindowClose(version string) bool {
	version = strings.TrimSpace(version)
	return version != "" && !versionLess(version, windowCloseMinVersion)
}

// CloseWindow removes one whole Chrome window. Closing the window that drives
// the current session is only allowed when another live session can take over
// first; otherwise the page would keep pointing at a window that no longer
// exists. Last-normal-window and extension capability checks happen before any
// job is sent.
func (m *Manager) CloseWindow(ctx context.Context, windowID int64, sessionID string) (*CloseWindowResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if windowID == 0 && sessionID == "" {
		return nil, errWindowCloseTarget
	}
	if windowID != 0 && sessionID != "" {
		return nil, errWindowCloseBoth
	}
	if sessionID != "" && strings.ContainsAny(sessionID, " \t\r\n/?#&=%") {
		return nil, errInvalidWindowSession
	}
	if m.isReadOnlyValue() {
		return nil, errReadOnly
	}

	// One gesture: an attach, close, or current-session switch must not race and
	// leave two operations pointing at the same windows.
	m.attachMu.Lock()
	defer m.attachMu.Unlock()

	rows, err := m.d.sessions(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := m.fetchWindowsForClose(ctx, rows)
	if err != nil {
		return nil, err
	}
	if !payload.supportsAllWindows() || !payload.supportsWindowClose() {
		return nil, errWindowCloseUnsupported
	}

	target, targetSession, err := resolveCloseTarget(payload, windowID, sessionID)
	if err != nil {
		return nil, err
	}
	normal := 0
	for i := range payload.Windows {
		if payload.Windows[i].Type == "" || payload.Windows[i].Type == "normal" {
			normal++
		}
	}
	if normal <= 1 {
		return nil, errWindowCloseLast
	}

	current := m.sessionIDValue()
	currentWindow := m.windowIDValue()
	targetCurrent := closeTargetIsCurrent(target, current, currentWindow)
	before := targetSessions(target, current, currentWindow)

	anchor := ""
	fallback := ""
	if targetCurrent {
		fallback = pickCloseFallback(current, rows, payload, before, target.WindowID)
		if fallback == "" {
			return nil, errWindowCloseCurrent
		}
		anchor = fallback
	} else {
		anchor = pickCloseAnchor(current, rows, payload, before)
		if anchor == "" {
			return nil, errWindowCloseNoAnchor
		}
	}

	params := map[string]any{"op": "close", "window_id": target.WindowID}
	if _, err := m.jobIn(ctx, anchor, 0, "window", params, 20*time.Second); err != nil {
		return nil, fmt.Errorf("close Chrome window %d: %w", target.WindowID, err)
	}

	result := &CloseWindowResult{WindowID: target.WindowID, SessionID: targetSession}
	if targetCurrent {
		if err := m.SwitchSession(ctx, fallback, false); err != nil {
			// The window is gone even if the takeover failed. Say so plainly so the
			// page refreshes instead of pretending the old current still exists.
			return result, fmt.Errorf("window %d closed, but switching to %s failed: %w", target.WindowID, fallback, err)
		}
		result.Current = fallback
	}
	return result, nil
}

// fetchWindowsForClose asks a close-capable live session first. A stale anchor
// running 1.0.21 would answer with a feature list that cannot authorize close.
func (m *Manager) fetchWindowsForClose(ctx context.Context, rows []sessionRow) (*windowsPayload, error) {
	anchor := pickWindowCloseAnchor(m.sessionIDValue(), rows, m.connectedValue())
	if anchor == "" {
		anchor = pickAllWindowsAnchor(m.sessionIDValue(), rows, m.connectedValue())
	}
	if anchor == "" {
		return nil, errNoLiveWindow
	}
	return m.fetchWindows(ctx, anchor)
}

func pickWindowCloseAnchor(current string, rows []sessionRow, connected bool) string {
	live := make([]string, 0, len(rows))
	versions := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.SessionID == "" || !row.Extension.Connected || row.SessionPageCount == 0 {
			continue
		}
		versions[row.SessionID] = row.Extension.Version
		live = append(live, row.SessionID)
	}
	sort.SliceStable(live, func(i, j int) bool {
		ci := extensionSupportsWindowClose(versions[live[i]])
		cj := extensionSupportsWindowClose(versions[live[j]])
		if ci != cj {
			return ci
		}
		if (live[i] == current) != (live[j] == current) {
			return live[i] == current
		}
		return live[i] < live[j]
	})
	for _, id := range live {
		if extensionSupportsWindowClose(versions[id]) {
			return id
		}
	}
	if connected && extensionSupportsWindowClose(versions[current]) {
		return current
	}
	return ""
}

// resolveCloseTarget maps exactly one selector to a window. Discovery only lists
// normal windows, so a found row is necessarily closable on Chrome's side.
func resolveCloseTarget(payload *windowsPayload, windowID int64, sessionID string) (*windowSnapshot, string, error) {
	if windowID != 0 {
		win := payload.windowByID(windowID)
		if win == nil {
			return nil, "", errWindowGone
		}
		targetSession := ""
		if len(win.SessionIDs) > 0 {
			targetSession = win.SessionIDs[0]
		}
		return win, targetSession, nil
	}

	win := payload.windowForSession(sessionID)
	if win == nil && isSessionlessSessionID(sessionID) {
		if id, ok := windowIDFromSessionlessID(sessionID); ok {
			win = payload.windowByID(id)
			if win == nil {
				return nil, "", errWindowGone
			}
			return win, sessionID, nil
		}
		return nil, "", errInvalidWindowSession
	}
	if win == nil {
		return nil, "", errWindowGone
	}
	return win, sessionID, nil
}

// closeTargetIsCurrent recognizes both a real session and a discovered window
// whose reserved synthetic id is current.
func closeTargetIsCurrent(target *windowSnapshot, current string, currentWindow int64) bool {
	if currentWindow != 0 && target.WindowID == currentWindow {
		return true
	}
	if isSessionlessSessionID(current) {
		if id, ok := windowIDFromSessionlessID(current); ok && id == target.WindowID {
			return true
		}
	}
	for _, id := range target.SessionIDs {
		if id == current {
			return true
		}
	}
	return false
}

func targetSessions(target *windowSnapshot, current string, currentWindow int64) map[string]bool {
	out := make(map[string]bool, len(target.SessionIDs)+1)
	for _, id := range target.SessionIDs {
		out[id] = true
	}
	if closeTargetIsCurrent(target, current, currentWindow) && current != "" {
		out[current] = true
	}
	return out
}

// pickCloseFallback returns a live session outside target. It becomes both the
// close transport and the session driven after the window disappears.
func pickCloseFallback(current string, rows []sessionRow, payload *windowsPayload, excluded map[string]bool, targetWindow int64) string {
	candidates := make([]string, 0, len(rows))
	versions := make(map[string]string, len(rows))
	for _, row := range rows {
		id := row.SessionID
		if id == "" || id == current || excluded[id] || !row.Extension.Connected || row.SessionPageCount == 0 {
			continue
		}
		versions[id] = row.Extension.Version
		candidates = append(candidates, id)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		ci := extensionSupportsWindowClose(versions[candidates[i]])
		cj := extensionSupportsWindowClose(versions[candidates[j]])
		if ci != cj {
			return ci
		}
		return candidates[i] < candidates[j]
	})
	for _, id := range candidates {
		if !extensionSupportsWindowClose(versions[id]) {
			continue
		}
		win := payload.windowForSession(id)
		if win == nil || win.WindowID == targetWindow {
			continue
		}
		return id
	}
	return ""
}

// pickCloseAnchor prefers the current session when it is live and outside the
// target, then any other live close-capable session. Closing a non-current
// window never needs a fallback: the current one survives.
func pickCloseAnchor(current string, rows []sessionRow, payload *windowsPayload, excluded map[string]bool) string {
	if !excluded[current] && mSessionLiveIn(rows, current) && mSessionCloseCapable(rows, current) {
		if win := payload.windowForSession(current); win != nil {
			return current
		}
	}
	return pickCloseFallback("", rows, payload, excluded, 0)
}

func mSessionLiveIn(rows []sessionRow, id string) bool {
	for _, row := range rows {
		if row.SessionID == id && row.Extension.Connected && row.SessionPageCount > 0 {
			return true
		}
	}
	return false
}

func mSessionCloseCapable(rows []sessionRow, id string) bool {
	for _, row := range rows {
		if row.SessionID == id {
			return extensionSupportsWindowClose(row.Extension.Version)
		}
	}
	return false
}

func (m *Manager) isReadOnlyValue() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readOnly
}

func (m *Manager) connectedValue() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}
