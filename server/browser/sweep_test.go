package browser

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const sweepSessionURL = "http://127.0.0.1:43799/go?session="

func sweepTab(id int64, target string) windowTab {
	return windowTab{TabID: id, URL: target}
}

func sweepWindow(id int64, focused bool, sessions []string, tabs ...windowTab) windowSnapshot {
	return windowSnapshot{WindowID: id, Type: "normal", Focused: focused, SessionIDs: sessions, Tabs: tabs}
}

func sweepPayload(open ...windowSnapshot) *windowsPayload {
	return &windowsPayload{
		Type:     "windows",
		Version:  "1.0.22",
		Features: []string{"all-windows", "window-close"},
		Windows:  open,
	}
}

func newSweepFake(rows []sessionRow, payload *windowsPayload) *fakeClient {
	data, _ := json.Marshal(payload)
	f := &fakeClient{rows: rows}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "windows":
			return json.RawMessage(data), true
		case "create_tab":
			return json.RawMessage(`{"tab_id":950}`), true
		}
		return nil, false
	}
	return f
}

// sweepRows is the usual three-window browser: the current session in 900 with
// two content tabs, a second session in 910, and a plain window in 920.
func sweepRows() []sessionRow { return closeRows("1.0.22", "s1", "s2", "s3") }

func sweepBrowser() []windowSnapshot {
	return []windowSnapshot{
		sweepWindow(900, true, []string{"s1"},
			sweepTab(901, sweepSessionURL+"s1"), sweepTab(902, "https://a.example/"), sweepTab(903, "https://b.example/")),
		sweepWindow(910, false, []string{"s2"},
			sweepTab(911, sweepSessionURL+"s2"), sweepTab(912, "https://c.example/")),
		sweepWindow(920, false, nil, sweepTab(921, "https://d.example/")),
	}
}

func TestSweepDryRunPlansWithoutClosing(t *testing.T) {
	f := newSweepFake(sweepRows(), sweepPayload(sweepBrowser()...))
	m := newCloseManager(t, f)

	result, err := m.SweepWindows(context.Background(), true)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if !result.DryRun {
		t.Fatal("dry_run = false")
	}
	if result.KeptWindowID != 900 || result.KeptSessionID != "s1" {
		t.Fatalf("keeper = %d/%q, want 900/s1", result.KeptWindowID, result.KeptSessionID)
	}
	if result.KeepTabs != 1 {
		t.Fatalf("keep_tabs = %d, want 1", result.KeepTabs)
	}
	// 910 and 920 hold no agent page, so both go; 900 keeps the session page.
	if result.CloseWindows != 2 {
		t.Fatalf("close_windows = %d, want 2", result.CloseWindows)
	}
	// 902, 903 in the keeper, 911, 912 in 910, 921 in 920.
	if result.CloseTabs != 5 {
		t.Fatalf("close_tabs = %d, want 5", result.CloseTabs)
	}
	if len(result.ClosedWindows) != 0 || result.CreatedBlank || result.ClosedTabs != 0 {
		t.Fatalf("dry run applied something: %+v", result)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("dry run closed a window")
	}
	if _, ok := f.last("cdp"); ok {
		t.Fatal("dry run closed a tab")
	}
	if _, ok := f.last("create_tab"); ok {
		t.Fatal("dry run opened a tab")
	}
}

func TestSweepClosesKeeperTabsThenWindowsThenOpensBlank(t *testing.T) {
	f := newSweepFake(sweepRows(), sweepPayload(sweepBrowser()...))
	m := newCloseManager(t, f)

	result, err := m.SweepWindows(context.Background(), false)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if result.CreatedBlank != true {
		t.Fatal("created_blank = false")
	}
	if result.ClosedTabs != 5 {
		t.Fatalf("closed_tabs = %d, want 5", result.ClosedTabs)
	}
	if len(result.ClosedWindows) != 2 || result.ClosedWindows[0] != 910 || result.ClosedWindows[1] != 920 {
		t.Fatalf("closed_windows = %v, want [910 920]", result.ClosedWindows)
	}

	// Keeper tabs go through the CDP tab close, anchored inside the keeper.
	tabs := f.all("cdp")
	if len(tabs) != 2 {
		t.Fatalf("cdp calls = %d, want 2", len(tabs))
	}
	for i, want := range []string{"902", "903"} {
		if got := tabs[i].Params["method"]; got != "Target.closeTarget" {
			t.Fatalf("cdp method = %v", got)
		}
		inner, _ := tabs[i].Params["params"].(map[string]any)
		if got := inner["targetId"]; got != want {
			t.Fatalf("targetId = %v, want %s", got, want)
		}
		if tabs[i].Session != "s1" {
			t.Fatalf("tab close anchor = %q, want s1", tabs[i].Session)
		}
	}

	windows := f.all("window")
	if len(windows) != 2 {
		t.Fatalf("window calls = %d, want 2", len(windows))
	}
	for i, want := range []int64{910, 920} {
		if got := windows[i].Params["op"]; got != "close" {
			t.Fatalf("op = %v", got)
		}
		if got := windows[i].Params["window_id"]; got != want {
			t.Fatalf("window_id = %v, want %d", got, want)
		}
	}

	created := f.all("create_tab")
	if len(created) != 1 {
		t.Fatalf("create_tab calls = %d, want 1", len(created))
	}
	if got := created[0].Params["url"]; got != "about:blank" {
		t.Fatalf("url = %v", got)
	}
	if got := created[0].Params["window_id"]; got != int64(900) {
		t.Fatalf("window_id = %v, want the keeper 900", got)
	}
	if got := created[0].Params["active"]; got != true {
		t.Fatalf("active = %v, want true", got)
	}

	// The sweep must not switch the session the person is watching.
	if m.sessionIDValue() != "s1" {
		t.Fatalf("current session = %q, want s1", m.sessionIDValue())
	}
}

func TestSweepKeepsCurrentSessionWindowEvenWhenAnotherIsFocused(t *testing.T) {
	open := sweepBrowser()
	open[0].Focused = false
	open[1].Focused = true
	f := newSweepFake(sweepRows(), sweepPayload(open...))
	m := newCloseManager(t, f)

	result, err := m.SweepWindows(context.Background(), true)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if result.KeptWindowID != 900 {
		t.Fatalf("kept = %d, want the current session's 900", result.KeptWindowID)
	}
	if len(result.ClosedWindows) != 0 {
		t.Fatalf("closed_windows set on a dry run: %v", result.ClosedWindows)
	}
}

func TestSweepFallsBackToFocusedWindow(t *testing.T) {
	// The current session's page is gone, so the payload has no window for it.
	// The focused window must be the one kept, and the sweep must not pretend
	// the manager's current session is its transport.
	open := []windowSnapshot{
		sweepWindow(900, false, nil, sweepTab(902, "https://a.example/")),
		sweepWindow(910, true, []string{"s2"},
			sweepTab(911, sweepSessionURL+"s2"), sweepTab(912, "https://c.example/")),
	}
	f := newSweepFake(closeRows("1.0.22", "s2"), sweepPayload(open...))
	m := newCloseManager(t, f)
	m.sessionID = "s1"

	result, err := m.SweepWindows(context.Background(), false)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if result.KeptWindowID != 910 || result.KeptSessionID != "s2" {
		t.Fatalf("keeper = %d/%q, want 910/s2", result.KeptWindowID, result.KeptSessionID)
	}
	if len(result.ClosedWindows) != 1 || result.ClosedWindows[0] != 900 {
		t.Fatalf("closed_windows = %v, want [900]", result.ClosedWindows)
	}
	for _, call := range f.all("window") {
		if call.Session != "s2" {
			t.Fatalf("close anchor = %q, want s2", call.Session)
		}
	}
	if m.sessionIDValue() != "s1" {
		t.Fatalf("current session = %q, want s1", m.sessionIDValue())
	}
}

func TestSweepSkipsDevtoolsWindows(t *testing.T) {
	open := sweepBrowser()
	open = append(open, windowSnapshot{WindowID: 930, Type: "devtools",
		Tabs: []windowTab{sweepTab(931, "devtools://devtools/bundled/inspector.html")}})
	f := newSweepFake(sweepRows(), sweepPayload(open...))
	m := newCloseManager(t, f)

	result, err := m.SweepWindows(context.Background(), false)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if result.CloseTabs != 5 {
		t.Fatalf("close_tabs = %d, want 5 (devtools excluded)", result.CloseTabs)
	}
	for _, call := range f.all("window") {
		if call.Params["window_id"] == int64(930) {
			t.Fatal("devtools window was closed")
		}
	}
}

func TestSweepRefusesSingleWindow(t *testing.T) {
	f := newSweepFake(closeRows("1.0.22", "s1"), sweepPayload(sweepBrowser()[0]))
	m := newCloseManager(t, f)

	if _, err := m.SweepWindows(context.Background(), true); !errors.Is(err, errSweepOneWindow) {
		t.Fatalf("err = %v, want one window", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestSweepRefusesOldExtension(t *testing.T) {
	payload := sweepPayload(sweepBrowser()...)
	payload.Version = "1.0.21"
	payload.Features = []string{"all-windows"}
	f := newSweepFake(sweepRows(), payload)
	m := newCloseManager(t, f)

	if _, err := m.SweepWindows(context.Background(), true); !errors.Is(err, errWindowCloseUnsupported) {
		t.Fatalf("err = %v, want unsupported", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestSweepNeedsALiveAnchorInTheKeeper(t *testing.T) {
	// Two windows and a live session, but no window lists it: nothing can carry
	// the jobs, so the sweep must refuse instead of closing the wrong window.
	f := newSweepFake(closeRows("1.0.22", "s1"), sweepPayload(
		sweepWindow(900, true, nil, sweepTab(902, "https://a.example/")),
		sweepWindow(910, false, nil, sweepTab(912, "https://c.example/")),
	))
	m := newCloseManager(t, f)

	if _, err := m.SweepWindows(context.Background(), true); !errors.Is(err, errSweepNoAnchor) {
		t.Fatalf("err = %v, want no anchor", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestSweepNeedsKeeperRegistrationPage(t *testing.T) {
	// The keeper's /go page is gone: nothing is left to keep and no transport.
	open := []windowSnapshot{
		sweepWindow(900, true, []string{"s1"}, sweepTab(902, "https://a.example/")),
		sweepWindow(910, false, []string{"s2"}, sweepTab(911, sweepSessionURL+"s2")),
	}
	f := newSweepFake(closeRows("1.0.22", "s1", "s2"), sweepPayload(open...))
	m := newCloseManager(t, f)

	if _, err := m.SweepWindows(context.Background(), true); !errors.Is(err, errSweepNoAnchor) {
		t.Fatalf("err = %v, want no anchor", err)
	}
}

func TestSweepReadOnly(t *testing.T) {
	f := newSweepFake(sweepRows(), sweepPayload(sweepBrowser()...))
	m := newCloseManager(t, f)
	m.readOnly = true

	if _, err := m.SweepWindows(context.Background(), false); !errors.Is(err, errReadOnly) {
		t.Fatalf("err = %v, want read only", err)
	}
}

func TestSweepToleratesAWindowClosedUnderIt(t *testing.T) {
	// 910 disappears between the plan and its close. That is not a failure: the
	// sweep continues with 920 and still ends on one window with a blank page.
	open := sweepBrowser()
	f := &fakeClient{rows: sweepRows()}
	full, _ := json.Marshal(sweepPayload(open...))
	reduced, _ := json.Marshal(sweepPayload(open[0], open[2]))
	fetches := 0
	f.fail = func(call fakeCall) error {
		if call.Type == "window" && call.Params["window_id"] == int64(910) {
			return errors.New("that Chrome window is already closed")
		}
		return nil
	}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		switch call.Type {
		case "windows":
			// The first listing is the plan; the second proves 910 is gone.
			fetches++
			if fetches > 1 {
				return json.RawMessage(reduced), true
			}
			return json.RawMessage(full), true
		case "create_tab":
			return json.RawMessage(`{"tab_id":950}`), true
		}
		return nil, false
	}

	m := newCloseManager(t, f)
	result, err := m.SweepWindows(context.Background(), false)
	if err != nil {
		t.Fatalf("SweepWindows: %v", err)
	}
	if result.CreatedBlank != true {
		t.Fatal("blank page was not opened")
	}
	if len(result.ClosedWindows) != 2 {
		t.Fatalf("closed_windows = %v, want both", result.ClosedWindows)
	}
}

func TestSweepReportsPartialFailure(t *testing.T) {
	// 910 refuses to close and is still listed: stop and report what did close.
	f := newSweepFake(sweepRows(), sweepPayload(sweepBrowser()...))
	f.fail = func(call fakeCall) error {
		if call.Type == "window" && call.Params["window_id"] == int64(910) {
			return errors.New("cannot close the window")
		}
		return nil
	}
	m := newCloseManager(t, f)

	result, err := m.SweepWindows(context.Background(), false)
	if err == nil {
		t.Fatal("SweepWindows succeeded, want the close failure")
	}
	if !strings.Contains(err.Error(), "910") {
		t.Fatalf("err = %v, want the window id", err)
	}
	if result == nil {
		t.Fatal("no partial result")
	}
	if result.ClosedTabs != 2 {
		t.Fatalf("closed_tabs = %d, want the keeper's 2", result.ClosedTabs)
	}
	if len(result.ClosedWindows) != 0 || result.CreatedBlank {
		t.Fatalf("partial result claims more than it did: %+v", result)
	}
	if _, ok := f.last("create_tab"); ok {
		t.Fatal("blank page opened after a failed close")
	}
}
