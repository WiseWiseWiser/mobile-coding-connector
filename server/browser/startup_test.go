package browser

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPassiveEnsureNeverLaunchesFromStalePageCount(t *testing.T) {
	f := healthyFake()
	f.sessionFn = func(id string) (*sessionRow, error) { return &sessionRow{SessionID: id, SessionPageCount: 1}, nil }
	m := newTestManager(t, f)
	m.connected, m.windowID, m.contentTabID = true, 900, 902
	opened := 0
	m.opener = func(context.Context, string) error { opened++; return nil }
	err := m.Ensure(context.Background())
	if !errors.Is(err, errBrowserOffline) || opened != 0 {
		t.Fatalf("error=%v opens=%d", err, opened)
	}
	if s := m.Snapshot(); s.Session.Connected || s.Session.WindowID != 0 || len(s.Tabs) != 0 {
		t.Fatalf("stale live snapshot: %+v", s)
	}
}

func TestExplicitStartIgnoresStalePageCountAndDeduplicates(t *testing.T) {
	f := healthyFake()
	var connected atomic.Bool
	f.sessionFn = func(id string) (*sessionRow, error) {
		return &sessionRow{SessionID: id, SessionPageCount: 1, Extension: extensionInfo{Connected: connected.Load()}}, nil
	}
	m := newTestManager(t, f)
	entered, release := make(chan struct{}), make(chan struct{})
	var opens atomic.Int32
	m.opener = func(_ context.Context, url string) error {
		if !strings.Contains(url, "/go?session=ai-critic") {
			t.Errorf("unexpected URL %s", url)
		}
		opens.Add(1)
		close(entered)
		<-release
		connected.Store(true)
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- m.StartBrowser(context.Background()) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("opener not called")
	}
	if err := m.StartBrowser(context.Background()); !errors.Is(err, errBrowserStarting) {
		t.Fatalf("concurrent error: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if opens.Load() != 1 || !m.Snapshot().Session.Connected || m.Snapshot().Startup != "ready" {
		t.Fatalf("opens=%d status=%+v", opens.Load(), m.Snapshot())
	}
}

func TestConnectedExtensionReattachesWithoutOpeningChrome(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	opens := 0
	m.opener = func(context.Context, string) error { opens++; return nil }
	m.pollTabs(context.Background())
	if opens != 0 || !m.Snapshot().Session.Connected {
		t.Fatalf("opens=%d status=%+v", opens, m.Snapshot())
	}
}

func TestStartFailureIsVisibleAndDoesNotRetry(t *testing.T) {
	f := healthyFake()
	f.sessionFn = func(id string) (*sessionRow, error) { return &sessionRow{SessionID: id, SessionPageCount: 1}, nil }
	m := newTestManager(t, f)
	opens := 0
	m.opener = func(context.Context, string) error { opens++; return errors.New("launch denied") }
	if err := m.StartBrowser(context.Background()); err == nil || !strings.Contains(err.Error(), "launch denied") {
		t.Fatalf("error = %v", err)
	}
	if opens != 1 || m.Snapshot().Startup != "failed" || !strings.Contains(m.Snapshot().Error, "launch denied") {
		t.Fatalf("opens=%d status=%+v", opens, m.Snapshot())
	}
}

func TestDisconnectedCurrentRowDoesNotPretendCachedWindowIsLive(t *testing.T) {
	f := healthyFake()
	f.rows = []sessionRow{{SessionID: "ai-critic", SessionPageCount: 1}}
	m := newTestManager(t, f)
	m.connected, m.windowID, m.contentTabID = true, 900, 902
	m.tabs = []TabInfo{{TabID: 902, URL: "https://example.com", Role: "user"}}
	result := m.Sessions(context.Background())
	if len(result.Sessions) != 1 {
		t.Fatalf("rows: %+v", result.Sessions)
	}
	row := result.Sessions[0]
	if row.Attached || row.Attachable || row.WindowID != 0 || len(row.Tabs) != 0 {
		t.Fatalf("offline row advertises a window: %+v", row)
	}
}
