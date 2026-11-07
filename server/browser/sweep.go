package browser

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Sweeping collapses every Chrome window into the one that holds the current
// session, so a person can reclaim the memory of a dozen windows in one gesture.
//
// It needs no extension capability of its own: a sweep is a "windows" listing,
// one "window close" per other window, a CDP tab close inside the keeper, then
// one "create_tab". Everything already exists for 1.0.22+, so the feature gates
// are the window-close gates.
//
// The keeper is the window hosting the current session's /go page. Its
// registration pages stay (the extension answers jobs through them), the rest of
// its tabs go, every other drivable window goes, and one active blank page opens
// last so the mirror lands on a cheap page instead of the last tab closed.
var (
	errSweepOneWindow = errors.New("only one Chrome window is open; nothing to close")
	errSweepNoAnchor  = errors.New("cannot keep a Chrome window without a live browser-agent page; open its /go page first")
)

// SweepWindowsResult is the dry-run plan and the applied outcome in one shape.
// A dry run fills the planned counts and leaves the applied fields empty, so the
// client can show exact numbers in its confirmation before anything closes.
type SweepWindowsResult struct {
	DryRun        bool    `json:"dry_run"`
	KeptWindowID  int64   `json:"kept_window_id"`
	KeptSessionID string  `json:"kept_session_id,omitempty"`
	KeepTabs      int     `json:"keep_tabs"`
	CloseTabs     int     `json:"close_tabs"`
	CloseWindows  int     `json:"close_windows"`
	ClosedWindows []int64 `json:"closed_windows,omitempty"`
	ClosedTabs    int     `json:"closed_tabs"`
	CreatedBlank  bool    `json:"created_blank"`
}

// sweepPlan is the resolved intention: which window survives, which session
// drives the jobs, and what leaves.
type sweepPlan struct {
	keeper *windowSnapshot
	anchor string
	// closeWindows are the drivable windows to remove, in listing order.
	closeWindows []int64
	// closeTabs are the keeper's non-registration tabs.
	closeTabs []int64
	// keepTabs counts the keeper's /go pages; they are the transport.
	keepTabs int
	// closeTabTotal counts every tab that leaves: the keeper's extra tabs plus
	// the tabs of the closed windows. It is the number the confirmation names.
	closeTabTotal int
}

// drivableWindow reports whether a window is a rail row: a normal, popup or app
// window the extension can address. DevTools and tooling windows are listed for
// context but neither counted nor closed.
func drivableWindow(w *windowSnapshot) bool {
	switch w.Type {
	case "", "normal", "popup", "app":
		return true
	}
	return false
}

// SweepWindows keeps one window and closes everything else. With dryRun it only
// reports the plan. Failures stop the sweep and return what was already closed:
// silently continuing would report a state the person cannot verify.
func (m *Manager) SweepWindows(ctx context.Context, dryRun bool) (*SweepWindowsResult, error) {
	if m.isReadOnlyValue() {
		return nil, errReadOnly
	}

	// One gesture: a sweep must not interleave with an attach, a close or a
	// current-session switch that is also rearranging windows.
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
	plan, err := buildSweepPlan(payload, rows, m.sessionIDValue())
	if err != nil {
		return nil, err
	}

	result := &SweepWindowsResult{
		DryRun:        dryRun,
		KeptWindowID:  plan.keeper.WindowID,
		KeptSessionID: plan.anchor,
		KeepTabs:      plan.keepTabs,
		CloseTabs:     plan.closeTabTotal,
		CloseWindows:  len(plan.closeWindows),
	}
	if dryRun {
		return result, nil
	}

	closedTabs := 0
	// fail reports the partial sweep: the client names what did close.
	fail := func(err error) (*SweepWindowsResult, error) {
		result.ClosedTabs = closedTabs
		return result, err
	}

	// The keeper's own tabs first: each close is a CDP call the extension allows
	// only for tabs of the anchor session's window, which is the keeper by
	// construction.
	for _, tabID := range plan.closeTabs {
		params := map[string]any{
			"method": "Target.closeTarget",
			"params": map[string]any{"targetId": strconv.FormatInt(tabID, 10)},
		}
		if _, err := m.jobIn(ctx, plan.anchor, 0, "cdp", params, 15*time.Second); err != nil {
			if !m.sweepTargetGone(ctx, plan.anchor, plan.keeper.WindowID, tabID) {
				return fail(fmt.Errorf("close tab %d in window %d: %w", tabID, plan.keeper.WindowID, err))
			}
		}
		closedTabs++
	}

	for _, windowID := range plan.closeWindows {
		params := map[string]any{"op": "close", "window_id": windowID}
		if _, err := m.jobIn(ctx, plan.anchor, 0, "window", params, 20*time.Second); err != nil {
			if !m.sweepTargetGone(ctx, plan.anchor, windowID, 0) {
				return fail(fmt.Errorf("close Chrome window %d: %w", windowID, err))
			}
		}
		result.ClosedWindows = append(result.ClosedWindows, windowID)
		closedTabs += tabsInWindow(payload, windowID)
	}

	// The blank page is the mirror's landing spot. It is active on purpose: the
	// person asked to end up on an empty page, not on whichever tab Chrome
	// activated while the others closed.
	if _, err := m.createTabIn(ctx, plan.anchor, plan.keeper.WindowID, "about:blank", true); err != nil {
		return fail(fmt.Errorf("open the blank page in window %d: %w", plan.keeper.WindowID, err))
	}
	result.CreatedBlank = true
	result.ClosedTabs = closedTabs
	return result, nil
}

// sweepTargetGone reports whether a job's target disappeared on its own. Chrome
// closes windows and tabs under the daemon all the time; a sweep must not stop
// half-done because a window the person just closed is still in its plan.
func (m *Manager) sweepTargetGone(ctx context.Context, anchor string, windowID, tabID int64) bool {
	payload, err := m.fetchWindows(ctx, anchor)
	if err != nil {
		return false
	}
	if payload.windowByID(windowID) == nil {
		return true
	}
	if tabID == 0 {
		return false
	}
	for i := range payload.Windows {
		for _, tab := range payload.Windows[i].Tabs {
			if tab.TabID == tabID {
				return false
			}
		}
	}
	return true
}

// tabsInWindow is how many tabs a window loses when it closes.
func tabsInWindow(payload *windowsPayload, windowID int64) int {
	if win := payload.windowByID(windowID); win != nil {
		return len(win.Tabs)
	}
	return 0
}

// buildSweepPlan resolves the keeper and everything that must leave. It refuses
// a single-window browser (nothing to save) and a keeper whose registration page
// is gone (no transport, and no page worth keeping).
func buildSweepPlan(payload *windowsPayload, rows []sessionRow, current string) (*sweepPlan, error) {
	keeper, anchor, err := pickSweepKeeper(payload, rows, current)
	if err != nil {
		return nil, err
	}
	plan := &sweepPlan{keeper: keeper, anchor: anchor}
	for _, tab := range keeper.Tabs {
		if isSessionPageURL(tab.URL) {
			plan.keepTabs++
			continue
		}
		plan.closeTabs = append(plan.closeTabs, tab.TabID)
	}
	if plan.keepTabs == 0 {
		return nil, errSweepNoAnchor
	}
	total := len(plan.closeTabs)
	for i := range payload.Windows {
		win := &payload.Windows[i]
		if win.WindowID == keeper.WindowID || !drivableWindow(win) {
			continue
		}
		plan.closeWindows = append(plan.closeWindows, win.WindowID)
		total += len(win.Tabs)
	}
	if len(plan.closeWindows) == 0 {
		return nil, errSweepOneWindow
	}
	plan.closeTabTotal = total
	return plan, nil
}

// pickSweepKeeper prefers the window that hosts the current session, so the page
// the person is watching survives and stays current. Without one it keeps the
// focused window, then the first window that holds a registration page.
func pickSweepKeeper(payload *windowsPayload, rows []sessionRow, current string) (*windowSnapshot, string, error) {
	if current != "" && mSessionLiveIn(rows, current) {
		if win := payload.windowForSession(current); win != nil && drivableWindow(win) {
			return win, current, nil
		}
	}
	// Focused first: the window the person is looking at is the least surprising
	// one to keep. A second pass ignores focus.
	for _, focusedOnly := range []bool{true, false} {
		for i := range payload.Windows {
			win := &payload.Windows[i]
			if !drivableWindow(win) || (focusedOnly && !win.Focused) {
				continue
			}
			if anchor := sweepAnchor(win, rows, current); anchor != "" {
				return win, anchor, nil
			}
		}
	}
	return nil, "", errSweepNoAnchor
}

// sweepAnchor returns a live session that runs inside win, so every sweep job
// rides the window that is being kept. The current session wins when it is there,
// then the lowest id, so repeated sweeps pick the same transport.
func sweepAnchor(win *windowSnapshot, rows []sessionRow, current string) string {
	live := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.SessionID == "" || !row.Extension.Connected || row.SessionPageCount == 0 {
			continue
		}
		live[row.SessionID] = true
	}
	candidates := make([]string, 0, len(win.SessionIDs))
	for _, id := range win.SessionIDs {
		if live[id] {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		isCurrentI := candidates[i] == current
		isCurrentJ := candidates[j] == current
		if isCurrentI != isCurrentJ {
			return isCurrentI
		}
		return candidates[i] < candidates[j]
	})
	return candidates[0]
}
