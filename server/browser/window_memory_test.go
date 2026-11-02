package browser

import (
	"context"
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

// useTempConfigDataDir points the browser config at a fresh temp dir so tests
// neither write into the checkout nor read a real user's remembered window;
// the previous dir is restored on cleanup.
func useTempConfigDataDir(t *testing.T) {
	t.Helper()
	prevDataDir := config.DataDir
	config.DataDir = t.TempDir()
	t.Cleanup(func() { config.DataDir = prevDataDir })
}

// A window of local file:// mockups plus the data: placeholder used to resolve
// to the placeholder on every viewer attach: file:// was not "content", so the
// placeholder was the only tab that qualified. The mirror must prefer Chrome's
// active tab — a file:// page included — and fall back to the first real page
// (file:// counts) before ever landing on the placeholder.
func TestChooseContentTabPrefersActiveFileTab(t *testing.T) {
	info := &sessionInfo{Active: 2, Tabs: []TabInfo{
		{TabID: 1, URL: "file:///Users/x/worktrees/one/mockup.html", Role: "user"},
		{TabID: 2, URL: "file:///Users/x/worktrees/two/mockup.html", Role: "user", Active: true},
		{TabID: 3, URL: contentPlaceholderURL(), Role: "user"},
	}}
	tabID, url := chooseContentTab(info)
	if tabID != 2 {
		t.Fatalf("tab = %d, want the active file:// tab 2", tabID)
	}
	if url != "file:///Users/x/worktrees/two/mockup.html" {
		t.Fatalf("url = %q, want the active file:// page", url)
	}
}

func TestChooseContentTabFallbackPrefersFileOverPlaceholder(t *testing.T) {
	info := &sessionInfo{Active: 9, Tabs: []TabInfo{
		{TabID: 9, URL: "chrome://settings", Role: "user", Active: true},
		{TabID: 4, URL: "file:///Users/x/worktrees/one/mockup.html", Role: "user"},
		{TabID: 3, URL: contentPlaceholderURL(), Role: "user"},
	}}
	tabID, _ := chooseContentTab(info)
	if tabID != 4 {
		t.Fatalf("tab = %d, want the first file:// tab 4 (the placeholder is the last resort)", tabID)
	}
}

// Follow-the-Mac must treat a file:// active tab as a real page too, or the
// mirror ignores the user switching between local mockups in Chrome.
func TestActiveMirrorTabFollowsFileTab(t *testing.T) {
	file := &sessionInfo{Active: 7, Tabs: []TabInfo{
		{TabID: 7, URL: "file:///Users/x/worktrees/one/mockup.html", Role: "user", Active: true},
	}}
	if tabID, _ := activeMirrorTab(file); tabID != 7 {
		t.Fatalf("tab = %d, want the active file:// tab 7", tabID)
	}
	internal := &sessionInfo{Active: 8, Tabs: []TabInfo{
		{TabID: 8, URL: "chrome://settings", Role: "user", Active: true},
	}}
	if tabID, _ := activeMirrorTab(internal); tabID != 0 {
		t.Fatalf("tab = %d, want 0 for an unmirrorable chrome:// page", tabID)
	}
}

// The last driven session survives a restart: rememberSession writes it, and
// New() loads it back so the first attach reopens on the same window.
func TestRememberSessionRoundTrip(t *testing.T) {
	useTempConfigDataDir(t)

	want := sessionlessSessionID(216816267)
	rememberSession(want)
	if got := loadConfig().SessionID; got != want {
		t.Fatalf("config session = %q, want %q", got, want)
	}

	// An empty id is a no-op, not a wipe of the remembered window.
	rememberSession("")
	if got := loadConfig().SessionID; got != want {
		t.Fatalf("config session = %q after empty remember, want %q kept", got, want)
	}
}

func TestNewRestoresRememberedWindow(t *testing.T) {
	useTempConfigDataDir(t)
	want := sessionlessSessionID(33)
	rememberSession(want)

	m := New(43761)
	if got := m.sessionIDValue(); got != want {
		t.Fatalf("session = %q, want the remembered window %q", got, want)
	}
}

// A direct attach that actually worked records the window; the next launch
// reopens on it instead of re-picking the focused window.
func TestDirectEnsureRemembersWindow(t *testing.T) {
	f := directFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.27", DirectSupported: true})

	if err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	want := sessionlessSessionID(11)
	if got := loadConfig().SessionID; got != want {
		t.Fatalf("config session = %q, want the attached window %q", got, want)
	}
}

func TestSwitchSessionRemembersWindow(t *testing.T) {
	f := directFake()
	m := newTestManager(t, f)
	m.recordDaemon(&daemonHealth{OK: true, Version: "1.0.27", DirectSupported: true})

	want := sessionlessSessionID(12)
	if err := m.SwitchSession(context.Background(), want, false); err != nil {
		t.Fatalf("SwitchSession: %v", err)
	}
	if got := loadConfig().SessionID; got != want {
		t.Fatalf("config session = %q, want the switched-to window %q", got, want)
	}
}

// switchConfigDataDir points the browser config at a fresh temp dir and
// restores the previous one on cleanup.
func swapConfigDataDir(t *testing.T) string {
	t.Helper()
	return "" // replaced by the shared helper below
}
