package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Tab operations. They map one-to-one onto the Chrome extension's cdp
// polyfills (Target.activateTarget / Target.closeTarget / create_tab).
const (
	TabActivate = "activate"
	TabClose    = "close"
	TabCreate   = "create"
)

// minTabDaemonVersion is the first browser-agent daemon whose extension has the
// Target.activateTarget / Target.closeTarget polyfills. Older daemons can still
// stream and take input, but the rail must not offer tab controls.
const minTabDaemonVersion = "1.0.20"

// tabInfoTTL keeps the rail honest without job spam: the capture loop refreshes
// the tab list on this cadence while somebody is watching.
const tabInfoTTL = 1500 * time.Millisecond

var (
	// errLastTab refuses to close the only content tab left. A window whose only
	// tab is the /go page has nothing to mirror; add a tab first.
	errLastTab = errors.New("cannot close the last tab: open a new tab first")
	// errSessionPage refuses to close the /go page: it is how the extension
	// knows which session (and window) it belongs to.
	errSessionPage = errors.New("cannot close the session page: it registers this Chrome window")
	// errReadOnly means the daemon is too old for tab multiplexing.
	errReadOnly = fmt.Errorf("tab controls need browser-agent %s+ (restart the daemon to upgrade)", minTabDaemonVersion)
)

// TabInfo is one Chrome tab as the daemon reports it.
type TabInfo struct {
	TabID  int64  `json:"tab_id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
	// Role is "session_page" for the /go tab and "user" for everything else.
	Role string `json:"role"`
}

func cloneTabs(in []TabInfo) []TabInfo {
	if len(in) == 0 {
		return nil
	}
	out := make([]TabInfo, len(in))
	copy(out, in)
	return out
}

// wireInfo is the daemon's info payload. The extension says `id`, the rest of
// this package says tab_id, so decode once here.
type wireInfo struct {
	WindowID int64 `json:"window_id"`
	Tabs     []struct {
		ID     int64  `json:"id"`
		TabID  int64  `json:"tab_id"`
		Title  string `json:"title"`
		URL    string `json:"url"`
		Active bool   `json:"active"`
		Role   string `json:"role"`
	} `json:"tabs"`
}

// sessionInfo is one session window: its id, window, tabs, and the tab Chrome
// currently shows there.
type sessionInfo struct {
	SessionID string
	WindowID  int64
	Tabs      []TabInfo
	// Active is the Chrome-active tab id. On a real window it is the session
	// page right after /go opens, until the user clicks a content tab.
	Active int64
}

// fetchInfo asks the extension for a session's window and tab list.
func (m *Manager) fetchInfo(ctx context.Context, sessionID string) (*sessionInfo, error) {
	data, err := m.jobIn(ctx, sessionID, 0, "info", nil, 15*time.Second)
	if err != nil {
		return nil, err
	}
	var wire wireInfo
	if err := json.Unmarshal(data, &wire); err != nil {
		return nil, fmt.Errorf("decode info: %w", err)
	}
	info := &sessionInfo{SessionID: sessionID, WindowID: wire.WindowID}
	for _, t := range wire.Tabs {
		id := t.ID
		if id == 0 {
			id = t.TabID
		}
		if t.Active {
			info.Active = id
		}
		role := t.Role
		if role == "" {
			if isSessionPageURL(t.URL) {
				role = "session_page"
			} else {
				role = "user"
			}
		}
		info.Tabs = append(info.Tabs, TabInfo{
			TabID:  id,
			Title:  strings.TrimSpace(t.Title),
			URL:    t.URL,
			Active: t.Active,
			Role:   role,
		})
	}
	return info, nil
}

// isSessionPageURL reports whether a tab is a /go?session= registration page.
func isSessionPageURL(raw string) bool {
	return strings.Contains(raw, "/go?session=")
}

// chooseContentTab picks the tab the mirror drives: the active tab when it is a
// real page, else the first usable one. The session page never qualifies: it is
// the registration anchor, and capturing it would be useless.
func chooseContentTab(info *sessionInfo) (tabID int64, url string) {
	var fallback int64
	var fallbackURL string
	for _, t := range info.Tabs {
		if t.Role == "session_page" || isSessionPageURL(t.URL) {
			continue
		}
		if !isContentURL(t.URL) {
			continue
		}
		if t.Active {
			return t.TabID, reportableURL(t.URL)
		}
		if fallback == 0 {
			fallback, fallbackURL = t.TabID, reportableURL(t.URL)
		}
	}
	return fallback, fallbackURL
}

// activeContentTab returns Chrome's active tab when it is a mirrorable page. A
// window sitting on the /go page returns 0: the user did not choose a content
// tab there, so the mirror must keep driving what it already had.
func activeContentTab(info *sessionInfo) (int64, string) {
	if info == nil {
		return 0, ""
	}
	t := tabOf(info, info.Active)
	if t.TabID == 0 || t.Role == "session_page" || isSessionPageURL(t.URL) || !isContentURL(t.URL) {
		return 0, ""
	}
	return t.TabID, reportableURL(t.URL)
}

// userTabs counts the tabs a person opened, ignoring the session page.
func userTabs(info *sessionInfo) int {
	n := 0
	for _, t := range info.Tabs {
		if t.Role == "session_page" || isSessionPageURL(t.URL) {
			continue
		}
		n++
	}
	return n
}

// applyInfo records a fresh info snapshot for sessionID. It is a no-op for a
// session the user has since switched away from, so a slow poll cannot drag the
// mirror back to the previous window.
func (m *Manager) applyInfo(sessionID string, info *sessionInfo) {
	if info == nil {
		return
	}
	m.mu.Lock()
	if m.sessionIDValueLocked() != sessionID {
		m.mu.Unlock()
		return
	}
	if info.WindowID != 0 {
		m.windowID = info.WindowID
	}
	m.tabs = cloneTabs(info.Tabs)
	m.activeTabID = info.Active
	m.tabsAt = time.Now()
	m.mu.Unlock()
}

// sessionIDValueLocked is sessionIDValue for callers already holding m.mu.
func (m *Manager) sessionIDValueLocked() string {
	if m.sessionID == "" {
		return SessionID
	}
	return m.sessionID
}

// pollTabs refreshes the tab list for the current session, follows the user's
// tab switches in Chrome, and re-points the mirror when the driven tab is gone.
// It runs from the capture loop, so it is throttled and best-effort.
func (m *Manager) pollTabs(ctx context.Context) {
	m.mu.Lock()
	sessionID := m.sessionIDValueLocked()
	stale := time.Since(m.tabsAt) > tabInfoTTL
	m.mu.Unlock()
	if !stale {
		return
	}
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return
	}
	m.applyInfo(sessionID, info)

	// Follow Chrome: if the user activated another content tab, drive that one.
	// A window parked on the /go tab leaves the mirror alone.
	if tabID, url := activeContentTab(info); tabID != 0 && tabID != m.contentTab() {
		m.mu.Lock()
		if m.sessionIDValueLocked() != sessionID {
			m.mu.Unlock()
			return
		}
		m.contentTabID = tabID
		if url != "" {
			m.url = url
		}
		m.viewportAt = time.Time{}
		m.mu.Unlock()
		_ = m.refreshViewport(ctx)
		m.hub.kick()
	}
}

// Tabs reports the cached tab list of the current session plus the tab the
// mirror is driving (what the strip highlights).
func (m *Manager) Tabs() ([]TabInfo, int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneTabs(m.tabs), m.drivenTabLocked()
}

// drivenTabLocked is the tab the mirror captures: the resolved content tab,
// falling back to Chrome's active tab before the first resolve.
func (m *Manager) drivenTabLocked() int64 {
	if m.contentTabID != 0 {
		return m.contentTabID
	}
	return m.activeTabID
}

// drivenTab is drivenTabLocked for callers not holding m.mu.
func (m *Manager) drivenTab() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.drivenTabLocked()
}

// TabOp activates, closes, or creates a tab. Every op re-reads the tab list
// from Chrome afterwards, because the extension's view is the only truth about
// what exists.
func (m *Manager) TabOp(ctx context.Context, sessionID, op string, tabID int64, rawURL string) (*sessionInfo, error) {
	if sessionID == "" {
		sessionID = m.sessionIDValue()
	}
	if m.isReadOnly() {
		return nil, errReadOnly
	}
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	switch op {
	case TabActivate:
		if err := m.tabExists(info, tabID); err != nil {
			return nil, err
		}
		if _, err := m.jobIn(ctx, sessionID, 0, "cdp", map[string]any{
			"method": "Target.activateTarget",
			"params": map[string]any{"targetId": strconv.FormatInt(tabID, 10)},
		}, 15*time.Second); err != nil {
			return nil, fmt.Errorf("activate tab: %w", err)
		}
		return m.retarget(ctx, sessionID, tabID)

	case TabClose:
		if err := m.tabExists(info, tabID); err != nil {
			return nil, err
		}
		if isSessionPageURL(tabOf(info, tabID).URL) {
			return nil, errSessionPage
		}
		if userTabs(info) <= 1 {
			return nil, errLastTab
		}
		if _, err := m.jobIn(ctx, sessionID, 0, "cdp", map[string]any{
			"method": "Target.closeTarget",
			"params": map[string]any{"targetId": strconv.FormatInt(tabID, 10)},
		}, 15*time.Second); err != nil {
			return nil, fmt.Errorf("close tab: %w", err)
		}
		// Chrome reports the closed tab for a moment after the ack; poll briefly
		// so the response already reflects reality.
		for i := 0; i < 6; i++ {
			time.Sleep(120 * time.Millisecond)
			if fresh, err := m.fetchInfo(ctx, sessionID); err == nil {
				if fresh.isGone(tabID) {
					info = fresh
					break
				}
				info = fresh
			}
		}
		return m.retarget(ctx, sessionID, 0)

	case TabCreate:
		target := strings.TrimSpace(rawURL)
		if target == "" {
			target = contentPlaceholderURL()
		} else if !isContentURL(target) && !strings.HasPrefix(target, "chrome://") {
			target = normalizeURL(target)
		}
		windowID := info.WindowID
		if windowID == 0 {
			return nil, fmt.Errorf("the browser-agent extension reported no window for this session")
		}
		created, err := m.createTabIn(ctx, sessionID, windowID, target)
		if err != nil {
			return nil, err
		}
		return m.retarget(ctx, sessionID, created)

	default:
		return nil, fmt.Errorf("unknown tab op %q", op)
	}
}

// retarget re-reads the tab list and, for the current session, moves the mirror
// to want (or to the best remaining tab when want is 0) and refreshes geometry.
func (m *Manager) retarget(ctx context.Context, sessionID string, want int64) (*sessionInfo, error) {
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	m.applyInfo(sessionID, info)

	if m.sessionIDValue() != sessionID {
		return info, nil
	}
	target := want
	if target == 0 || !info.has(target) {
		target, _ = chooseContentTab(info)
	}
	m.mu.Lock()
	changed := m.contentTabID != target
	m.contentTabID = target
	if tab := tabOf(info, target); tab.TabID != 0 {
		if u := reportableURL(tab.URL); u != "" {
			m.url = u
		}
	}
	m.viewportAt = time.Time{}
	m.mu.Unlock()
	if changed && target != 0 {
		_ = m.refreshViewport(ctx)
	}
	m.hub.kick()
	m.broadcastStatus()
	return info, nil
}

// createTabIn opens a content tab in a session's window.
func (m *Manager) createTabIn(ctx context.Context, sessionID string, windowID int64, target string) (int64, error) {
	data, err := m.jobIn(ctx, sessionID, 0, "create_tab", map[string]any{
		"url":       target,
		"window_id": windowID,
		"active":    true,
	}, 20*time.Second)
	if err != nil {
		return 0, err
	}
	var created struct {
		TabID int64 `json:"tab_id"`
		ID    int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		return 0, fmt.Errorf("decode create_tab: %w", err)
	}
	if created.TabID != 0 {
		return created.TabID, nil
	}
	if created.ID != 0 {
		return created.ID, nil
	}
	return 0, fmt.Errorf("create_tab returned no tab id")
}

func (m *Manager) tabExists(info *sessionInfo, tabID int64) error {
	if tabID == 0 {
		return fmt.Errorf("tab id is required")
	}
	if !info.has(tabID) {
		return fmt.Errorf("tab %d is not in this session", tabID)
	}
	return nil
}

func (info *sessionInfo) has(tabID int64) bool {
	for _, t := range info.Tabs {
		if t.TabID == tabID {
			return true
		}
	}
	return false
}

func (info *sessionInfo) isGone(tabID int64) bool { return !info.has(tabID) }

// tabOf returns a tab or the zero value. Callers check TabID != 0.
func tabOf(info *sessionInfo, tabID int64) TabInfo {
	for _, t := range info.Tabs {
		if t.TabID == tabID {
			return t
		}
	}
	return TabInfo{}
}

func (m *Manager) isReadOnly() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readOnly
}

// SessionInfo is one row of the session rail: the session's window plus as much
// live state as the daemon will tell us about it.
type SessionInfo struct {
	ID       string `json:"id"`
	Current  bool   `json:"current"`
	Attached bool   `json:"attached"`
	// Attachable is false for an explicitly opened session whose Chrome window
	// is gone: the rail shows it dimmed with a Reopen button.
	Attachable  bool          `json:"attachable"`
	WindowID    int64         `json:"window_id,omitempty"`
	URL         string        `json:"url,omitempty"`
	Preset      string        `json:"preset,omitempty"`
	Mode        string        `json:"mode,omitempty"`
	Viewport    Viewport      `json:"viewport"`
	ActiveTabID int64         `json:"active_tab_id,omitempty"`
	Tabs        []TabInfo     `json:"tabs"`
	Capture     CaptureStatus `json:"capture"`
	Error       string        `json:"error,omitempty"`
}

// SessionsResponse is the body of GET /api/browser/sessions.
type SessionsResponse struct {
	OK       bool          `json:"ok"`
	Daemon   DaemonStatus  `json:"daemon"`
	Current  string        `json:"current"`
	Sessions []SessionInfo `json:"sessions"`
	ReadOnly bool          `json:"read_only,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// Sessions lists the sessions worth showing: those with a live Chrome window
// (extension attached and its /go page open) plus the current one. The daemon
// registry keeps hundreds of stale ids from earlier runs; none of them belong in
// a sidebar.
func (m *Manager) Sessions(ctx context.Context) SessionsResponse {
	current := m.sessionIDValue()
	m.mu.Lock()
	resp := SessionsResponse{
		OK:       true,
		Daemon:   DaemonStatus{Running: m.daemonVersion != "", Port: m.port, Version: m.daemonVersion},
		Current:  current,
		ReadOnly: m.readOnly,
	}
	m.mu.Unlock()

	rows, err := m.d.sessions(ctx)
	if err != nil {
		resp.OK = false
		resp.Error = err.Error()
		// Still report the current session so the page can render a rail even
		// when the daemon cannot enumerate.
		resp.Sessions = []SessionInfo{m.currentSessionInfo()}
		return resp
	}

	seen := map[string]bool{}
	for _, row := range rows {
		if row.SessionID == "" || seen[row.SessionID] {
			continue
		}
		live := row.Extension.Connected && row.SessionPageCount > 0
		if !live && row.SessionID != current {
			continue
		}
		seen[row.SessionID] = true
		resp.Sessions = append(resp.Sessions, SessionInfo{
			ID:         row.SessionID,
			Current:    row.SessionID == current,
			Attached:   row.Extension.Connected,
			Attachable: live,
		})
	}
	if !seen[current] {
		resp.Sessions = append(resp.Sessions, SessionInfo{
			ID:      current,
			Current: true,
		})
	}

	// Fetch each live session's tabs in parallel; a dead window still needs its
	// row, so errors are recorded per session instead of failing the list.
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range resp.Sessions {
		if !resp.Sessions[i].Attachable {
			continue
		}
		wg.Add(1)
		go func(si *SessionInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			info, err := m.fetchInfo(ctx, si.ID)
			if err != nil {
				si.Error = err.Error()
				return
			}
			si.WindowID = info.WindowID
			si.Tabs = cloneTabs(info.Tabs)
			si.ActiveTabID = info.Active
			if tabID, u := chooseContentTab(info); tabID != 0 {
				si.URL = u
			}
		}(&resp.Sessions[i])
	}
	wg.Wait()

	// The current session may also have been described from the cache; prefer
	// the live measurement the rest of the page already trusts.
	for i := range resp.Sessions {
		if !resp.Sessions[i].Current {
			continue
		}
		m.mu.Lock()
		resp.Sessions[i].Preset = m.preset
		resp.Sessions[i].Mode = m.mode
		resp.Sessions[i].Viewport = m.viewport
		resp.Sessions[i].Attached = m.connected
		resp.Sessions[i].Attachable = m.connected || len(m.tabs) > 0
		if len(m.tabs) > 0 {
			resp.Sessions[i].Tabs = cloneTabs(m.tabs)
			resp.Sessions[i].ActiveTabID = m.drivenTabLocked()
		}
		if resp.Sessions[i].WindowID == 0 {
			resp.Sessions[i].WindowID = m.windowID
		}
		if u := m.url; u != "" {
			resp.Sessions[i].URL = u
		}
		m.mu.Unlock()
	}
	if m.hub != nil {
		stats := m.hub.stats()
		for i := range resp.Sessions {
			if resp.Sessions[i].Current {
				resp.Sessions[i].Capture = stats
			}
		}
	}
	// Current first, then newest windows; the rail reads top-down.
	sort.SliceStable(resp.Sessions, func(i, j int) bool {
		if resp.Sessions[i].Current != resp.Sessions[j].Current {
			return resp.Sessions[i].Current
		}
		return resp.Sessions[i].ID < resp.Sessions[j].ID
	})
	return resp
}

func (m *Manager) currentSessionInfo() SessionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	si := SessionInfo{
		ID:          m.sessionIDValueLocked(),
		Current:     true,
		Attached:    m.connected,
		Attachable:  m.connected,
		WindowID:    m.windowID,
		URL:         m.url,
		Preset:      m.preset,
		Mode:        m.mode,
		Viewport:    m.viewport,
		ActiveTabID: m.drivenTabLocked(),
		Tabs:        cloneTabs(m.tabs),
	}
	if m.hub != nil {
		si.Capture = m.hub.stats()
	}
	return si
}

// SwitchSession makes another session current. Everything that decides which
// window we drive is reset, then Ensure re-registers the session page and picks
// the tab to mirror; the rail's Reopen button is the same call with newWindow.
func (m *Manager) SwitchSession(ctx context.Context, sessionID string, newWindow bool) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.ContainsAny(sessionID, " \t\r\n/?#&=%") {
		return fmt.Errorf("invalid session id %q", sessionID)
	}
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()

	m.mu.Lock()
	prev := m.sessionIDValueLocked()
	m.sessionID = sessionID
	m.connected = false
	m.windowID = 0
	m.contentTabID = 0
	m.activeTabID = 0
	m.tabs = nil
	m.tabsAt = time.Time{}
	m.url = ""
	m.viewport = Viewport{}
	m.viewportAt = time.Time{}
	m.macKnown = false
	m.lastOpenAt = time.Time{}
	m.lastErr = ""
	if newWindow {
		m.newWindowFor = sessionID
	}
	m.mu.Unlock()

	cfg := loadConfig()
	if cfg.SessionID != sessionID {
		cfg.SessionID = sessionID
		_ = saveConfig(cfg)
	}
	m.hub.kick()

	if err := m.ensureLocked(ctx); err != nil {
		m.setError(err)
		return err
	}
	_ = prev
	m.broadcastStatus()
	return nil
}

// NewSession mints a fresh session id and opens its page in a new Chrome
// window. The extension cannot create windows, so the host browser does it.
func (m *Manager) NewSession(ctx context.Context) (string, error) {
	id, err := m.nextSessionID(ctx)
	if err != nil {
		return "", err
	}
	if err := m.SwitchSession(ctx, id, true); err != nil {
		return id, err
	}
	return id, nil
}

// nextSessionID picks the first free ai-critic-<n> name.
func (m *Manager) nextSessionID(ctx context.Context) (string, error) {
	taken := map[string]bool{}
	if rows, err := m.d.sessions(ctx); err == nil {
		for _, row := range rows {
			taken[row.SessionID] = true
		}
	}
	next := 0
	for id := range taken {
		if !strings.HasPrefix(id, SessionID+"-") {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(id, SessionID+"-")); err == nil && n > next {
			next = n
		}
	}
	if next < 2 {
		next = 2
	}
	return fmt.Sprintf("%s-%d", SessionID, next), nil
}

// broadcastStatus pushes the new state to every browser page, so a tab op in one
// viewer's rail is visible in the others without waiting for their next poll.
func (m *Manager) broadcastStatus() {
	if m.hub == nil {
		return
	}
	m.hub.broadcastJSON(map[string]any{"type": "status", "status": m.Snapshot()})
}
