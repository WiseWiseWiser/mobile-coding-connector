package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	// errReadOnly means the daemon is too old for tab multiplexing.
	errReadOnly = fmt.Errorf("tab controls need browser-agent %s+ (restart the daemon to upgrade)", minTabDaemonVersion)
)

// TabInfo is one Chrome tab as the daemon reports it.
type TabInfo struct {
	TabID int64 `json:"tab_id"`
	// Index is Chrome's tab index within the window; it decides which page names
	// the window. It is optional because the info job of older extensions does
	// not report it.
	Index  int    `json:"index,omitempty"`
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
	WindowID int64    `json:"window_id"`
	Features []string `json:"features"`
	Tabs     []struct {
		ID     int64  `json:"id"`
		TabID  int64  `json:"tab_id"`
		Index  int    `json:"index"`
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
	// Features is the extension capability list reported by the info job.
	Features          []string
	CloseConfirmation *CloseTabConfirmation
	CloseOutcome      string
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
	info := &sessionInfo{SessionID: sessionID, WindowID: wire.WindowID, Features: wire.Features}
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
			Index:  t.Index,
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

// chooseMirrorTab is chooseContentTab with one relaxation for re-aiming the
// mirror on a window that is already being driven: when no other page is left,
// the /go anchor page is a real page too and the mirror follows it instead of
// freezing on a dead tab. Fresh attach (resolveTargets) and window claiming
// (failover) keep using chooseContentTab, which never drives the anchor.
func chooseMirrorTab(info *sessionInfo) (tabID int64, url string) {
	tabID, url = chooseContentTab(info)
	if tabID != 0 {
		return tabID, url
	}
	// No content page: follow the active tab, the /go anchor included.
	// chrome:// and about:blank stay unmirrorable.
	for _, t := range info.Tabs {
		if !t.Active {
			continue
		}
		if !isContentURL(t.URL) && !isSessionPageURL(t.URL) {
			return 0, ""
		}
		return t.TabID, reportableURL(t.URL)
	}
	return 0, ""
}

// activeMirrorTab returns Chrome's active tab when the mirror can drive it:
// any real page, the /go anchor included (chooseMirrorTab semantics).
// window sitting on the /go page returns 0: the user did not choose a content
// tab there, so the mirror must keep driving what it already had.
func activeMirrorTab(info *sessionInfo) (int64, string) {
	if info == nil {
		return 0, ""
	}
	t := tabOf(info, info.Active)
	if t.TabID == 0 {
		return 0, ""
	}
	if !isContentURL(t.URL) && !isSessionPageURL(t.URL) {
		return 0, ""
	}
	return t.TabID, reportableURL(t.URL)
}

// applyInfo records a fresh info snapshot for sessionID. It is a no-op for a
// session the user has since switched away from, so a slow poll cannot drag the
// mirror back to the previous window.
func (m *Manager) applyInfo(sessionID string, info *sessionInfo) bool {
	if info == nil {
		return false
	}
	m.mu.Lock()
	if m.sessionIDValueLocked() != sessionID {
		m.mu.Unlock()
		return false
	}
	changed := !slices.Equal(m.tabs, info.Tabs) || m.activeTabID != info.Active || (info.WindowID != 0 && m.windowID != info.WindowID)
	if info.WindowID != 0 {
		m.windowID = info.WindowID
	}
	m.tabs = cloneTabs(info.Tabs)
	m.activeTabID = info.Active
	m.tabsAt = time.Now()
	m.mu.Unlock()
	return changed
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
	// Direct rail: sessionless ids have no session row; the info job itself
	// proves the window is alive (a gone window fails and marks disconnect).
	if _, isSessionless := windowIDFromSessionlessID(sessionID); isSessionless && m.directSupportedValue() {
		m.mu.Lock()
		connected := m.connected
		m.mu.Unlock()
		if !connected {
			if m.Ensure(ctx) == nil {
				m.broadcastStatus()
			}
			return
		}
		info, err := m.fetchInfo(ctx, sessionID)
		if err != nil {
			m.markDisconnected(sessionID)
			m.mu.Lock()
			m.tabsAt = time.Now()
			m.mu.Unlock()
			return
		}
		m.finishPollTabs(sessionID, info)
		return
	}
	row, err := m.d.session(ctx, sessionID)
	if err != nil {
		return
	}
	if !row.Extension.Connected {
		m.markDisconnected(sessionID)
		m.mu.Lock()
		m.tabsAt = time.Now()
		m.mu.Unlock()
		return
	}
	m.mu.Lock()
	connected := m.connected
	m.mu.Unlock()
	if !connected {
		if m.Ensure(ctx) == nil {
			m.broadcastStatus()
		}
		return
	}
	info, err := m.fetchInfo(ctx, sessionID)
	if err != nil {
		return
	}
	m.finishPollTabs(sessionID, info)
}

// finishPollTabs applies a fresh info snapshot and follows the user's tab
// switches; shared by the session and direct poll paths.
func (m *Manager) finishPollTabs(sessionID string, info *sessionInfo) {
	ctx := context.Background()
	changed := m.applyInfo(sessionID, info)

	// Follow Chrome: if the user activated another page, drive that one — the
	// /go anchor included, it is a real page. A tab that cannot be captured
	// (chrome://) leaves the mirror alone.
	if tabID, url := activeMirrorTab(info); tabID != 0 && tabID != m.contentTab() {
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
	// The driven tab may navigate or redirect without changing its ID. Keep the
	// address in sync with Chrome, including clearing internal placeholder URLs.
	// The anchor page syncs too once the mirror drives it (see chooseMirrorTab).
	m.mu.Lock()
	if m.sessionIDValueLocked() == sessionID {
		tab := tabOf(info, m.contentTabID)
		if tab.TabID != 0 && (isContentURL(tab.URL) || isSessionPageURL(tab.URL)) {
			url := reportableURL(tab.URL)
			if m.url != url {
				m.url = url
				changed = true
			}
		}
	}
	m.mu.Unlock()
	if changed {
		m.broadcastStatus()
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
		// Window-id rail: every tab closes like any tab. The /go anchor is the
		// daemon's session registration and its close job refuses to touch it,
		// so anchor closes go over CDP, which has no such opinion. The
		// extension's confirmation flow still guards pages it considers
		// stateful.
		if !isSessionPageURL(tabOf(info, tabID).URL) && featureEnabled(info.Features, tabCloseConfirmationFeature) {
			result, err := m.remoteTabCloseJob(ctx, sessionID, "BrowserAgent.closeTab", map[string]any{"tab_id": tabID})
			if err != nil {
				if isTabCloseTimeout(err) {
					return nil, unknownTabCloseError()
				}
				return nil, fmt.Errorf("close tab: %w", err)
			}
			if result.TabID != tabID {
				return nil, fmt.Errorf("invalid tab close response target")
			}
			switch result.Status {
			case "confirmation_required":
				if result.OperationID == "" || result.TabID != tabID {
					return nil, fmt.Errorf("invalid tab close confirmation")
				}
				info.CloseConfirmation = &CloseTabConfirmation{OperationID: result.OperationID, TabID: tabID}
				return info, nil
			case "closed":
			case "unknown":
				return nil, unknownTabCloseError()
			default:
				return nil, fmt.Errorf("invalid tab close response: %q", result.Status)
			}
		} else if _, err := m.jobIn(ctx, sessionID, 0, "cdp", map[string]any{
			"method": "Target.closeTarget",
			"params": map[string]any{"targetId": strconv.FormatInt(tabID, 10)},
		}, 15*time.Second); err != nil {
			if isTabCloseTimeout(err) {
				return nil, unknownTabCloseError()
			}
			return nil, fmt.Errorf("close tab: %w", err)
		}
		// Chrome reports the closed tab for a moment after the ack; poll briefly
		// so the response already reflects reality. A failing info job right
		// after the ack means the closed tab was the last one and Chrome took
		// the window with it — drop the mirror state now; the poll's reconnect
		// fails over to a free window.
		windowGone := false
		for i := 0; i < 6; i++ {
			time.Sleep(120 * time.Millisecond)
			fresh, err := m.fetchInfo(ctx, sessionID)
			if err != nil {
				windowGone = true
				break
			}
			info = fresh
			if fresh.isGone(tabID) {
				break
			}
		}
		if windowGone || len(info.Tabs) == 0 {
			m.markDisconnected(sessionID)
			return info, nil
		}
		return m.retargetAfterClose(ctx, sessionID, info)

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
// retargetAfterClose re-aims the mirror after a tab close. Closing the last
// tab closes the whole window in Chrome, and closing the /go anchor ends the
// daemon session — either way the followup info job can fail. That must not
// turn the successful close into an error: the poll's reconnect re-aims or
// fails over on its own.
func (m *Manager) retargetAfterClose(ctx context.Context, sessionID string, info *sessionInfo) (*sessionInfo, error) {
	fresh, err := m.retarget(ctx, sessionID, 0)
	if err != nil {
		return info, nil
	}
	return fresh, nil
}

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
		target, _ = chooseMirrorTab(info)
	}
	m.mu.Lock()
	if m.sessionIDValueLocked() != sessionID {
		m.mu.Unlock()
		return info, nil
	}
	changed := m.contentTabID != target
	m.contentTabID = target
	if tab := tabOf(info, target); tab.TabID != 0 {
		m.url = reportableURL(tab.URL)
	} else {
		m.url = ""
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
	Attachable bool `json:"attachable"`
	// Title is the window label: the title of its first real page, not the
	// active tab. Sessionless windows are discovered windows the extension can
	// see but that no session is driving yet.
	Title       string `json:"title,omitempty"`
	TitleTabID  int64  `json:"title_tab_id,omitempty"`
	Sessionless bool   `json:"sessionless,omitempty"`
	// Adoptable means an explicit Attach can register this window without
	// opening a second Chrome window.
	Adoptable   bool          `json:"adoptable,omitempty"`
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
	// WindowsSupported is true when the connected extension can enumerate every
	// window (feature "all-windows"). Older extensions keep the rail as-is.
	WindowsSupported bool `json:"windows_supported"`
	// WindowCloseSupported is true only when the same extension also advertises
	// "window-close". Listing a window does not imply that it may be removed.
	WindowCloseSupported bool   `json:"window_close_supported"`
	ReadOnly             bool   `json:"read_only,omitempty"`
	Error                string `json:"error,omitempty"`
}

// Sessions lists the sessions worth showing: those with a live Chrome window
// (extension attached and its /go page open) plus the current one. The daemon
// registry keeps hundreds of stale ids from earlier runs; none of them belong in
// a sidebar. When the extension supports "all-windows" the list also gains one
// read-only row for every normal Chrome window that no session drives yet.
func (m *Manager) Sessions(ctx context.Context) SessionsResponse {
	current := m.sessionIDValue()
	m.mu.Lock()
	resp := SessionsResponse{
		OK:       true,
		Daemon:   DaemonStatus{Running: m.daemonVersion != "", Port: m.port, Version: m.daemonVersion},
		Current:  current,
		ReadOnly: m.readOnly,
	}
	currentConnected := m.connected
	currentPreset := m.preset
	currentMode := m.mode
	currentViewport := m.viewport
	currentTabs := cloneTabs(m.tabs)
	currentActive := m.drivenTabLocked()
	currentWindow := m.windowID
	currentURL := m.url
	seenWin := map[string]bool{}
	if currentWindow != 0 {
		seenWin[sessionlessSessionID(currentWindow)] = true
	}
	directCurrentConnected := m.connected
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

	// Direct rail: when the daemon supports window addressing, discovered
	// Chrome windows become attachable rows (sess-window-<id>) beside the
	// session rows. A job failure here only removes the window rows.
	// A window whose /go session page is alive is already represented by its
	// session row below, so it must not get a sess-window-<id> row too — one
	// window twice in the rail, with a phantom row whose × closes the real
	// window. Windows whose registered sessions are dead stay adoptable.
	liveSessions := map[string]bool{}
	for _, row := range rows {
		if row.Extension.Connected && row.SessionPageCount > 0 {
			liveSessions[row.SessionID] = true
		}
	}
	if m.directSupportedValue() {
		if payload, werr := m.d.windowsDirect(ctx); werr == nil {
			for i := range payload.Windows {
				w := &payload.Windows[i]
				if w.Type != "normal" {
					continue
				}
				claimed := false
				for _, sid := range w.SessionIDs {
					if liveSessions[sid] {
						claimed = true
						break
					}
				}
				if claimed {
					continue
				}
				id := sessionlessSessionID(w.WindowID)
				if seenWin[id] {
					continue
				}
				seenWin[id] = true
				if id == current {
					continue // current row is seeded from the cache below
				}
				info := &sessionInfo{SessionID: id, WindowID: w.WindowID, Tabs: w.tabInfos()}
				activeTabID := int64(0)
				for _, t := range info.Tabs {
					if t.Active {
						activeTabID = t.TabID
						break
					}
				}
				info.Active = activeTabID
				row := SessionInfo{
					ID:         id,
					Attached:   true,
					Attachable: true,
					// Adoptable lets the drawer row's click attach the window
					// through the direct rail (POST /api/browser/window/attach).
					Adoptable:   true,
					Sessionless: true,
					WindowID:    w.WindowID,
					Tabs:        cloneTabs(info.Tabs),
					ActiveTabID: activeTabID,
				}
				if tabID, u := chooseContentTab(info); tabID != 0 {
					row.URL = u
				}
				resp.Sessions = append(resp.Sessions, row)
			}
		}
	}

	// A successful registry response is authoritative; absent means offline.
	currentConnected = false
	// Direct rail: a sessionless current id has no registry row; the manager's
	// connection state stands in for the extension check below.
	if _, isSessionless := windowIDFromSessionlessID(current); isSessionless && m.directSupportedValue() {
		currentConnected = directCurrentConnected
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.SessionID == "" || seen[row.SessionID] {
			continue
		}
		// Direct rail: a sess-window-<id> anchor session never earns its own
		// row. The windows enumeration already lists every drivable window;
		// when the window is closed but its /go anchor survives in another
		// window, this row would be a drivable ghost — switching to it fails
		// with "window not found" even though the daemon reports the anchor
		// as connected.
		if m.directSupportedValue() {
			if _, isSessionless := windowIDFromSessionlessID(row.SessionID); isSessionless {
				continue
			}
		}
		live := row.Extension.Connected && row.SessionPageCount > 0
		if row.SessionID == current {
			currentConnected = row.Extension.Connected
		}
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

	// Seed the current row from the manager cache before asking the daemon. This
	// keeps its window, tabs and title stable when the info job is slow or fails.
	for i := range resp.Sessions {
		if !resp.Sessions[i].Current {
			continue
		}
		if _, isSessionless := windowIDFromSessionlessID(resp.Sessions[i].ID); isSessionless {
			// The direct rail drives this window without a registered session,
			// so the row must look like a controlled session to the phone UI:
			// "sessionless" would lock the address bar and offer Attach again.
			if !m.directSupportedValue() {
				resp.Sessions[i].Sessionless = true
			}
			if resp.Sessions[i].WindowID == 0 {
				resp.Sessions[i].WindowID = currentWindow
			}
		}
		resp.Sessions[i].Preset = currentPreset
		resp.Sessions[i].Mode = currentMode
		resp.Sessions[i].Viewport = currentViewport
		resp.Sessions[i].Attached = currentConnected
		if !currentConnected {
			continue
		}
		resp.Sessions[i].Attachable = resp.Sessions[i].Attachable || len(currentTabs) > 0
		if len(currentTabs) > 0 {
			resp.Sessions[i].Tabs = cloneTabs(currentTabs)
			resp.Sessions[i].ActiveTabID = currentActive
		}
		if currentWindow != 0 {
			resp.Sessions[i].WindowID = currentWindow
		}
		if currentURL != "" {
			resp.Sessions[i].URL = currentURL
		}
	}
	for i := range resp.Sessions {
		if resp.Sessions[i].Title == "" {
			resp.Sessions[i].Title, resp.Sessions[i].TitleTabID = windowTitleFromTabs(resp.Sessions[i].Tabs)
		}
	}

	// Fetch each live session's tabs in parallel; a dead window still needs its
	// row, so errors are recorded per session instead of failing the list.
	// Synthesized direct window rows already carry their tabs from the windows
	// payload and are skipped.
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i := range resp.Sessions {
		if resp.ReadOnly || !resp.Sessions[i].Attachable {
			continue
		}
		if resp.Sessions[i].Sessionless && len(resp.Sessions[i].Tabs) > 0 {
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
			si.Title, si.TitleTabID = windowTitleFromTabs(si.Tabs)
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
		resp.Sessions[i].Attached = currentConnected
		if !currentConnected {
			m.mu.Unlock()
			continue
		}
		resp.Sessions[i].Attachable = resp.Sessions[i].Attachable || len(m.tabs) > 0
		// The info job is a live measurement; the manager cache only fills in
		// when that job could not describe the window. Overwriting here would
		// drop fresh tab titles (and the active tab) for cached ones.
		if len(resp.Sessions[i].Tabs) == 0 && len(m.tabs) > 0 {
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
	for i := range resp.Sessions {
		if resp.Sessions[i].Title == "" {
			resp.Sessions[i].Title, resp.Sessions[i].TitleTabID = windowTitleFromTabs(resp.Sessions[i].Tabs)
		}
	}

	// Discovery is best-effort: a failed "windows" job must never hide the live
	// sessions above. Read-only mode is another reason not to touch the daemon.
	if !resp.ReadOnly {
		if anchor := pickAllWindowsAnchor(current, rows, currentConnected); anchor != "" {
			if payload, err := m.fetchWindows(ctx, anchor); err == nil && payload.supportsAllWindows() {
				resp.WindowsSupported = true
				resp.WindowCloseSupported = payload.supportsWindowClose()
				m.mergeWindows(&resp, payload)
			}
		}
	}
	for i := range resp.Sessions {
		if resp.Sessions[i].Title == "" {
			resp.Sessions[i].Title, resp.Sessions[i].TitleTabID = windowTitleFromTabs(resp.Sessions[i].Tabs)
		}
	}

	if m.hub != nil {
		stats := m.hub.stats()
		for i := range resp.Sessions {
			if resp.Sessions[i].Current {
				resp.Sessions[i].Capture = stats
			}
		}
	}
	// Stable order, independent of which row is current. Sorting by activity or
	// recency makes the rail jump under the pointer every time someone switches
	// sessions, so the list only follows the session ids.
	sort.SliceStable(resp.Sessions, func(i, j int) bool {
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
	return m.switchSessionLocked(ctx, sessionID, newWindow)
}

// switchSessionLocked is SwitchSession with ensureMu already held. It is also
// the last step of window adoption, where a failure must roll the manager back
// to the session the person was already watching.
func (m *Manager) switchSessionLocked(ctx context.Context, sessionID string, newWindow bool) error {
	m.mu.Lock()
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
	// The new window may still be sized for another preset; apply the saved one
	// again so switching sessions cannot make the size switcher lie.
	m.presetApplied = false
	m.lastOpenAt = time.Time{}
	m.lastErr = ""
	if newWindow {
		m.newWindowFor = sessionID
	}
	m.mu.Unlock()

	m.hub.kick()

	if err := m.ensureLocked(ctx, true); err != nil {
		m.setError(err)
		return err
	}
	// Persist only after attachment succeeds, so a failed window creation
	// cannot replace the previously working session on the next restart.
	if !isSessionlessSessionID(sessionID) {
		cfg := loadConfig()
		if cfg.SessionID != sessionID {
			cfg.SessionID = sessionID
			_ = saveConfig(cfg)
		}
	}
	m.broadcastStatus()
	return nil
}

// managerSessionState is the per-session state SwitchSession resets. Adoption
// snapshots it so a failed attach leaves the previous session selected.
type managerSessionState struct {
	sessionID     string
	connected     bool
	windowID      int64
	contentTabID  int64
	activeTabID   int64
	tabs          []TabInfo
	tabsAt        time.Time
	url           string
	extPath       string
	viewport      Viewport
	viewportAt    time.Time
	macBounds     Bounds
	macKnown      bool
	presetApplied bool
	lastErr       string
	ensuredAt     time.Time
	lastOpenAt    time.Time
	newWindowFor  string
}

func (m *Manager) sessionStateLocked() managerSessionState {
	return managerSessionState{
		sessionID:     m.sessionID,
		connected:     m.connected,
		windowID:      m.windowID,
		contentTabID:  m.contentTabID,
		activeTabID:   m.activeTabID,
		tabs:          cloneTabs(m.tabs),
		tabsAt:        m.tabsAt,
		url:           m.url,
		extPath:       m.extPath,
		viewport:      m.viewport,
		viewportAt:    m.viewportAt,
		macBounds:     m.macBounds,
		macKnown:      m.macKnown,
		presetApplied: m.presetApplied,
		lastErr:       m.lastErr,
		ensuredAt:     m.ensuredAt,
		lastOpenAt:    m.lastOpenAt,
		newWindowFor:  m.newWindowFor,
	}
}

func (m *Manager) restoreSessionStateLocked(state managerSessionState) {
	m.sessionID = state.sessionID
	m.connected = state.connected
	m.windowID = state.windowID
	m.contentTabID = state.contentTabID
	m.activeTabID = state.activeTabID
	m.tabs = cloneTabs(state.tabs)
	m.tabsAt = state.tabsAt
	m.url = state.url
	m.extPath = state.extPath
	m.viewport = state.viewport
	m.viewportAt = state.viewportAt
	m.macBounds = state.macBounds
	m.macKnown = state.macKnown
	m.presetApplied = state.presetApplied
	m.lastErr = state.lastErr
	m.ensuredAt = state.ensuredAt
	m.lastOpenAt = state.lastOpenAt
	m.newWindowFor = state.newWindowFor
}

// NewSession mints a fresh session id and opens its page in a new Chrome
// window. The extension cannot create windows, so the host browser does it.
// On a direct-capable daemon the new window is a plain window driven by window
// id — no browser-agent session, hence no /go page in it.
func (m *Manager) NewSession(ctx context.Context) (string, error) {
	m.ensureMu.Lock()
	defer m.ensureMu.Unlock()
	if err := m.ensureDaemon(ctx); err != nil {
		return "", err
	}
	if m.DirectSupported() {
		return m.newDirectWindow(ctx)
	}
	id, err := m.nextSessionID(ctx)
	if err != nil {
		return "", err
	}
	// Reserve the ID separately from the idempotent ensure path; a conflict
	// must never be mistaken for a successfully created new window.
	for attempts := 0; ; attempts++ {
		err = m.d.createFreshSession(ctx, id)
		if !errors.Is(err, errSessionAlreadyExists) {
			break
		}
		if attempts >= 99 {
			return "", fmt.Errorf("could not reserve an unused browser session: %w", err)
		}
		n, parseErr := strconv.Atoi(strings.TrimPrefix(id, SessionID+"-"))
		if parseErr != nil || n == int(^uint(0)>>1) {
			return "", fmt.Errorf("no browser session ID available")
		}
		id = fmt.Sprintf("%s-%d", SessionID, n+1)
	}
	if err != nil {
		return "", fmt.Errorf("reserve new browser session: %w", err)
	}
	m.mu.Lock()
	previous := m.sessionStateLocked()
	m.mu.Unlock()
	// Compare against every discoverable window before touching the newly
	// attached one. Older extensions still protect the current known window.
	existing := map[int64]bool{}
	if previous.windowID != 0 {
		existing[previous.windowID] = true
	}
	if previous.connected {
		probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		windows, probeErr := m.fetchWindows(probeCtx, previous.sessionID)
		cancel()
		if probeErr == nil && windows.supportsAllWindows() {
			for _, window := range windows.Windows {
				if window.WindowID != 0 {
					existing[window.WindowID] = true
				}
			}
		}
	}
	m.mu.Lock()
	m.freshWindowRequired = true
	m.previousWindowIDs = existing
	m.mu.Unlock()
	if err := m.switchSessionLocked(ctx, id, true); err != nil {
		m.mu.Lock()
		m.restoreSessionStateLocked(previous)
		m.freshWindowRequired = false
		m.previousWindowIDs = nil
		m.mu.Unlock()
		m.setError(err)
		m.broadcastStatus()
		return id, err
	}
	m.mu.Lock()
	m.freshWindowRequired = false
	m.previousWindowIDs = nil
	m.mu.Unlock()
	return id, nil
}

// directNewWindowURL opens a directly-driven window with a capturable page.
// A bare new-tab page (chrome://) and about:blank are not capturable, so the
// rail would have nothing to mirror until the first real navigation.
const directNewWindowURL = "data:text/html;charset=utf-8,%3C!doctype%20html%3E%3Ctitle%3Eai-critic%3C%2Ftitle%3E"

// newDirectWindow opens a plain Chrome window and adopts it over the direct
// rail: no browser-agent session is minted, so the window never gets a /go
// page. The fresh window is told apart from everything already on screen by
// its window id. The returned id is the sessionless window id.
func (m *Manager) newDirectWindow(ctx context.Context) (string, error) {
	m.mu.Lock()
	previous := m.sessionStateLocked()
	m.mu.Unlock()

	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	before, err := m.d.windowsDirect(probeCtx)
	cancel()
	if err != nil {
		err = fmt.Errorf("enumerate windows: %w", err)
		m.setError(err)
		return "", err
	}
	// No all-windows feature check here: the daemon's /v1/windows response
	// does not carry the feature list, and DirectSupported() has already
	// established the direct rail.
	known := make(map[int64]bool, len(before.Windows))
	for _, w := range before.Windows {
		known[w.WindowID] = true
	}
	if err := m.windowOpener(ctx, directNewWindowURL); err != nil {
		err = fmt.Errorf("open a new Chrome window: %w", err)
		m.setError(err)
		return "", err
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		after, probeErr := m.d.windowsDirect(probeCtx)
		cancel()
		if probeErr == nil {
			for _, w := range after.Windows {
				if w.WindowID == 0 || known[w.WindowID] {
					continue
				}
				// Wait until the window carries a capturable tab: a freshly
				// registered window can show up before Chrome lists its first
				// tab, and adopting too early would spawn a placeholder tab
				// next to the opener page.
				capturable := false
				for _, tb := range w.Tabs {
					if tb.Capturable || isContentURL(tb.URL) {
						capturable = true
						break
					}
				}
				if !capturable {
					continue
				}
				// Adopt through the standard direct switch so state reset, tab
				// resolution, preset and viewport reuse one code path.
				directID := sessionlessSessionID(w.WindowID)
				if err := m.switchSessionLocked(ctx, directID, false); err != nil {
					m.mu.Lock()
					m.restoreSessionStateLocked(previous)
					m.mu.Unlock()
					m.setError(err)
					return "", err
				}
				return directID, nil
			}
		}
		if time.Now().After(deadline) {
			m.mu.Lock()
			m.restoreSessionStateLocked(previous)
			m.mu.Unlock()
			err := fmt.Errorf("new Chrome window did not appear")
			m.setError(err)
			return "", err
		}
		select {
		case <-ctx.Done():
			m.mu.Lock()
			m.restoreSessionStateLocked(previous)
			m.mu.Unlock()
			return "", ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
}

// nextSessionID picks the first free ai-critic-<n> name.
func (m *Manager) nextSessionID(ctx context.Context) (string, error) {
	rows, err := m.d.sessions(ctx)
	if err != nil {
		return "", fmt.Errorf("list browser sessions: %w", err)
	}
	taken := make(map[string]bool, len(rows))
	for _, row := range rows {
		taken[row.SessionID] = true
	}
	for n := 2; n > 0; n++ {
		id := fmt.Sprintf("%s-%d", SessionID, n)
		if !taken[id] {
			return id, nil
		}
	}
	return "", fmt.Errorf("no browser session ID available")
}

// broadcastStatus pushes the new state to every browser page, so a tab op in one
// viewer's rail is visible in the others without waiting for their next poll.
func (m *Manager) broadcastStatus() {
	if m.hub == nil {
		return
	}
	m.hub.broadcastJSON(map[string]any{"type": "status", "status": m.Snapshot()})
}
