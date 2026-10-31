package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// allWindowsFeature is the extension capability that makes a "windows" job
// possible. It is a feature flag, not a version check: an old extension and a
// future one that drops the job must both fall back to the session-only rail.
const allWindowsFeature = "all-windows"

// allWindowsMinVersion is only an anchor preference. The extension still
// answers with features, and that response remains the actual gate.
const allWindowsMinVersion = "1.0.21"

// sessionlessIDPrefix names a discovered window that no live session drives.
// The id is deterministic so a row does not jump while the person clicks
// Attach, and it is reserved: it is never written to browser-config.json.
const sessionlessIDPrefix = "sess-window-"

var (
	errAllWindowsUnsupported = errors.New("the browser-agent extension does not support all windows (1.0.21+ required); reload it from the extension directory")
	errNoLiveWindow          = errors.New("open a browser-agent session page in Chrome first")
	errWindowGone            = errors.New("that Chrome window is no longer open")
	errWindowHasSession      = errors.New("that Chrome window already has a session; refresh the session list")
	errInvalidWindowSession  = errors.New("invalid window session id")
)

// windowsPayload is what the extension's "windows" job returns. Width, height
// and state are informational; adoption reuses the existing window and never
// resizes it until the saved preset is applied by the normal switch path.
type windowsPayload struct {
	Type     string           `json:"type"`
	Version  string           `json:"version"`
	Features []string         `json:"features"`
	Windows  []windowSnapshot `json:"windows"`
}

type windowSnapshot struct {
	WindowID   int64       `json:"window_id"`
	Type       string      `json:"type"`
	Focused    bool        `json:"focused"`
	Width      int         `json:"width"`
	Height     int         `json:"height"`
	State      string      `json:"state"`
	Title      string      `json:"title"`
	TitleTabID int64       `json:"title_tab_id"`
	SessionIDs []string    `json:"session_ids"`
	Tabs       []windowTab `json:"tabs"`
}

type windowTab struct {
	TabID      int64  `json:"tab_id"`
	Index      int    `json:"index"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Active     bool   `json:"active"`
	Capturable bool   `json:"capturable"`
	Role       string `json:"role"`
}

func (p *windowsPayload) supportsAllWindows() bool {
	return featureEnabled(p.Features, allWindowsFeature)
}

func featureEnabled(features []string, want string) bool {
	for _, f := range features {
		if f == want {
			return true
		}
	}
	return false
}

func (p *windowsPayload) windowByID(windowID int64) *windowSnapshot {
	for i := range p.Windows {
		if p.Windows[i].WindowID == windowID {
			return &p.Windows[i]
		}
	}
	return nil
}

func (p *windowsPayload) windowForSession(sessionID string) *windowSnapshot {
	for i := range p.Windows {
		for _, id := range p.Windows[i].SessionIDs {
			if id == sessionID {
				return &p.Windows[i]
			}
		}
	}
	return nil
}

// fetchWindows asks the extension for every normal window. The job is routed
// through anchor (a live session) because the extension only answers sessions
// that have a registered /go page; the answer describes every Chrome window in
// that extension's profile, not only the anchor's window.
func (m *Manager) fetchWindows(ctx context.Context, anchor string) (*windowsPayload, error) {
	data, err := m.jobIn(ctx, anchor, 0, "windows", nil, 8*time.Second)
	if err != nil {
		return nil, err
	}
	var payload windowsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode windows: %w", err)
	}
	return &payload, nil
}

func (w *windowSnapshot) tabInfos() []TabInfo {
	tabs := make([]TabInfo, 0, len(w.Tabs))
	for _, t := range w.Tabs {
		role := t.Role
		if role == "" {
			role = "user"
			if sid := sessionIDFromPageURL(t.URL); sid != "" {
				role = "session_page"
			}
		}
		tabs = append(tabs, TabInfo{
			TabID:  t.TabID,
			Index:  t.Index,
			Title:  cleanWindowTitle(t.Title),
			URL:    t.URL,
			Active: t.Active,
			Role:   role,
		})
	}
	sort.SliceStable(tabs, func(i, j int) bool {
		return tabs[i].Index < tabs[j].Index
	})
	return tabs
}

func (w *windowSnapshot) sessionPageTab(sessionID string) int64 {
	for _, t := range w.Tabs {
		if sessionIDFromPageURL(t.URL) == sessionID {
			return t.TabID
		}
	}
	return 0
}

// windowLabel is the title shown for a window. The extension already derives it
// from the lowest-index real page; fall back to the tabs when an older payload
// omits the field, then to the session id when the window is only its /go page.
func windowLabel(w *windowSnapshot) (string, int64) {
	if title := cleanWindowTitle(w.Title); title != "" {
		return title, w.TitleTabID
	}
	return windowTitleFromTabs(w.tabInfos())
}

func cleanWindowTitle(title string) string {
	return strings.Join(strings.Fields(title), " ")
}

func windowTitleFromTabs(tabs []TabInfo) (string, int64) {
	ordered := cloneTabs(tabs)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Index < ordered[j].Index
	})
	for _, t := range ordered {
		if t.Role == "session_page" || sessionIDFromPageURL(t.URL) != "" {
			continue
		}
		if !isWindowTitlePageURL(t.URL) {
			continue
		}
		title := cleanWindowTitle(t.Title)
		if title == "" {
			title = fallbackWindowTitle(t.URL)
		}
		if title != "" {
			return title, t.TabID
		}
	}
	for _, t := range ordered {
		if t.Role == "session_page" || sessionIDFromPageURL(t.URL) != "" {
			continue
		}
		if title := cleanWindowTitle(t.Title); title != "" {
			return title, t.TabID
		}
	}
	for _, t := range ordered {
		if sid := sessionIDFromPageURL(t.URL); sid != "" {
			return sid, 0
		}
	}
	return "Untitled window", 0
}

func isWindowTitlePageURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https", "file":
		return true
	default:
		return false
	}
}

func fallbackWindowTitle(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if host := u.Hostname(); host != "" {
		return host
	}
	if u.Scheme == "file" {
		if base := path.Base(u.Path); base != "." && base != "/" {
			return base
		}
	}
	if u.Scheme != "" {
		return u.Scheme
	}
	return ""
}

// sessionIDFromPageURL returns the session registered by a /go page on a
// loopback host, or "" for a normal page. It mirrors the extension's parser so
// a stale or forged remote /go URL is never mistaken for a registration page.
func sessionIDFromPageURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	switch u.Hostname() {
	case "127.0.0.1", "localhost", "::1":
	default:
		return ""
	}
	if u.Path != "/go" {
		return ""
	}
	return u.Query().Get("session")
}

func sessionlessSessionID(windowID int64) string {
	return fmt.Sprintf("%s%d", sessionlessIDPrefix, windowID)
}

func isSessionlessSessionID(sessionID string) bool {
	return strings.HasPrefix(sessionID, sessionlessIDPrefix)
}

// windowIDFromSessionlessID parses the synthetic id back to its window.
func windowIDFromSessionlessID(sessionID string) (int64, bool) {
	if !isSessionlessSessionID(sessionID) {
		return 0, false
	}
	windowID, err := strconv.ParseInt(strings.TrimPrefix(sessionID, sessionlessIDPrefix), 10, 64)
	if err != nil || windowID <= 0 {
		return 0, false
	}
	return windowID, true
}

// liveAnchor picks a session the extension will answer for. A live session
// running an extension that can list windows wins over the current one: after
// an adopted window closes, the current synthetic session is stale and must not
// pin discovery to an older extension. Discovery cannot work with zero live
// sessions at all: there is no extension transport until a /go page connected.
func (m *Manager) liveAnchor(ctx context.Context) (string, error) {
	current := m.sessionIDValue()
	rows, err := m.d.sessions(ctx)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	connected := m.connected
	m.mu.Unlock()
	if anchor := pickAllWindowsAnchor(current, rows, connected); anchor != "" {
		return anchor, nil
	}
	return "", errNoLiveWindow
}

// pickAllWindowsAnchor prefers a live session whose extension version can
// serve the windows job. The feature list in the response remains the real
// gate; the version only decides which live session to ask first.
func pickAllWindowsAnchor(current string, rows []sessionRow, connected bool) string {
	versions := make(map[string]string, len(rows))
	live := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.SessionID == "" || !row.Extension.Connected || row.SessionPageCount == 0 {
			continue
		}
		versions[row.SessionID] = row.Extension.Version
		live = append(live, row.SessionID)
	}
	if len(live) > 0 {
		sort.SliceStable(live, func(i, j int) bool {
			ci := extensionSupportsAllWindows(versions[live[i]])
			cj := extensionSupportsAllWindows(versions[live[j]])
			if ci != cj {
				return ci
			}
			return live[i] < live[j]
		})
		if extensionSupportsAllWindows(versions[current]) {
			for _, id := range live {
				if id == current {
					return current
				}
			}
		}
		if extensionSupportsAllWindows(versions[live[0]]) {
			return live[0]
		}
	}
	return pickLiveAnchor(current, rows, connected)
}

func extensionSupportsAllWindows(version string) bool {
	version = strings.TrimSpace(version)
	return version != "" && !versionLess(version, allWindowsMinVersion)
}

func pickLiveAnchor(current string, rows []sessionRow, connected bool) string {
	for _, row := range rows {
		if row.SessionID == current && row.Extension.Connected && row.SessionPageCount > 0 {
			return current
		}
	}
	var live []string
	for _, row := range rows {
		if row.SessionID != "" && row.Extension.Connected && row.SessionPageCount > 0 {
			live = append(live, row.SessionID)
		}
	}
	if len(live) > 0 {
		sort.Strings(live)
		return live[0]
	}
	if connected {
		return current
	}
	return ""
}

func (m *Manager) sessionIsLive(ctx context.Context, sessionID string) bool {
	row, err := m.d.session(ctx, sessionID)
	if err != nil {
		return false
	}
	return row.Extension.Connected && row.SessionPageCount > 0
}

// AttachWindow adopts an existing Chrome window: it opens the session page in
// that exact window, waits for it to register, verifies the extension reports
// the same window, and only then switches the manager to it. No second Chrome
// window is created, so the same-window /go invariant is preserved.
func (m *Manager) AttachWindow(ctx context.Context, windowID int64, requested string) (string, error) {
	if windowID <= 0 {
		return "", fmt.Errorf("window_id is required")
	}
	sessionID := sessionlessSessionID(windowID)
	requested = strings.TrimSpace(requested)
	if requested != "" && requested != sessionID {
		return "", errInvalidWindowSession
	}

	// Direct rail: the window needs no /go registration at all — verify it
	// exists, then switch; ensureDirectWindow re-validates and resolves tabs.
	// The capability flag comes from /v1/health, which a fresh process has not
	// recorded yet, so probe it here (best-effort: a failed probe keeps the
	// session-track attach flow, which needs a live /go page anyway).
	if h, err := m.d.health(ctx); err == nil {
		m.recordDaemon(h)
	}
	if m.directSupportedValue() {
		payload, err := m.d.windowsDirect(ctx)
		if err != nil {
			return "", fmt.Errorf("list Chrome windows: %w", err)
		}
		if payload.windowByID(windowID) == nil {
			return "", errWindowGone
		}
		m.ensureMu.Lock()
		defer m.ensureMu.Unlock()
		m.mu.Lock()
		previous := m.sessionStateLocked()
		m.mu.Unlock()
		err = m.switchSessionLocked(ctx, sessionID, false)
		if err != nil {
			m.mu.Lock()
			m.restoreSessionStateLocked(previous)
			m.mu.Unlock()
			m.setError(err)
			m.broadcastStatus()
			return "", err
		}
		return sessionID, nil
	}

	m.attachMu.Lock()
	defer m.attachMu.Unlock()

	anchor, err := m.liveAnchor(ctx)
	if err != nil {
		return "", err
	}
	payload, err := m.fetchWindows(ctx, anchor)
	if err != nil {
		return "", fmt.Errorf("list Chrome windows: %w", err)
	}
	if !payload.supportsAllWindows() {
		return "", errAllWindowsUnsupported
	}
	target := payload.windowByID(windowID)
	if target == nil {
		return "", errWindowGone
	}
	for _, sid := range target.SessionIDs {
		if sid == "" || sid == sessionID {
			continue
		}
		if m.sessionIsLive(ctx, sid) {
			return "", errWindowHasSession
		}
	}

	// The daemon only accepts registration from a session it knows. Register the
	// synthetic id before opening /go: the page alone cannot create it, and a
	// rejected register leaves the tab open with no session behind it.
	if err := m.d.createSession(ctx, sessionID); err != nil {
		return "", fmt.Errorf("register the adopted session: %w", err)
	}

	// A retry may find the /go page already in the window; wait on it instead of
	// opening a duplicate. create_tab is window-aware in 1.0.21, so the page
	// lands in the adopted window and nowhere else.
	if target.sessionPageTab(sessionID) == 0 {
		if _, err := m.createTabIn(ctx, anchor, windowID, m.sessionPageURL(sessionID)); err != nil {
			return "", fmt.Errorf("open the session page in that window: %w", err)
		}
	}
	if err := m.waitForSessionPage(ctx, sessionID, 20*time.Second); err != nil {
		return "", fmt.Errorf("register the session page: %w", err)
	}
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("verify the adopted window: %w", err)
	}
	if info.WindowID != windowID {
		return "", fmt.Errorf("the session page registered in window %d instead of %d", info.WindowID, windowID)
	}

	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	m.mu.Lock()
	previous := m.sessionStateLocked()
	m.mu.Unlock()
	err = m.switchSessionLocked(ctx, sessionID, false)
	if err != nil {
		m.mu.Lock()
		m.restoreSessionStateLocked(previous)
		m.mu.Unlock()
		m.setError(err)
		m.broadcastStatus()
		return "", err
	}
	return sessionID, nil
}

func (m *Manager) waitForSessionPage(ctx context.Context, sessionID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		row, err := m.d.session(ctx, sessionID)
		if err == nil && row.Extension.Connected && row.SessionPageCount > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return fmt.Errorf("the browser-agent extension did not register the session page")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// mergeWindows reconciles the session rail with the extension's window list.
// Existing sessions keep their rows (even several sessions in one window); a
// normal window without a live session becomes one read-only row named after
// its first real page. Matching prefers the known window id, then the session
// ids the /go pages report, so a stale manager cache cannot duplicate a window.
func (m *Manager) mergeWindows(resp *SessionsResponse, payload *windowsPayload) {
	matched := make([]bool, len(resp.Sessions))
	occupied := map[int64]bool{}
	for i := range resp.Sessions {
		si := &resp.Sessions[i]
		if si.WindowID == 0 {
			continue
		}
		occupied[si.WindowID] = true
		if win := payload.windowByID(si.WindowID); win != nil {
			applyWindowToRow(si, win)
			matched[i] = true
		}
	}
	for i := range resp.Sessions {
		si := &resp.Sessions[i]
		if matched[i] || si.WindowID != 0 {
			continue
		}
		if win := payload.windowForSession(si.ID); win != nil {
			applyWindowToRow(si, win)
			occupied[win.WindowID] = true
			matched[i] = true
		}
	}
	for i := range payload.Windows {
		win := &payload.Windows[i]
		if win.WindowID == 0 || occupied[win.WindowID] {
			continue
		}
		title, titleTabID := windowLabel(win)
		tabs := win.tabInfos()
		si := SessionInfo{
			ID:          sessionlessSessionID(win.WindowID),
			Title:       title,
			TitleTabID:  titleTabID,
			Sessionless: true,
			Adoptable:   true,
			WindowID:    win.WindowID,
			Tabs:        tabs,
		}
		for _, t := range tabs {
			if t.Active {
				si.ActiveTabID = t.TabID
				break
			}
		}
		if _, u := chooseContentTab(&sessionInfo{Tabs: tabs, Active: si.ActiveTabID}); u != "" {
			si.URL = u
		}
		resp.Sessions = append(resp.Sessions, si)
		occupied[win.WindowID] = true
	}
}

func applyWindowToRow(si *SessionInfo, win *windowSnapshot) {
	si.Title, si.TitleTabID = windowLabel(win)
	si.WindowID = win.WindowID
	if len(si.Tabs) == 0 {
		si.Tabs = win.tabInfos()
		if si.ActiveTabID == 0 {
			for _, t := range si.Tabs {
				if t.Active {
					si.ActiveTabID = t.TabID
					break
				}
			}
		}
	}
}
