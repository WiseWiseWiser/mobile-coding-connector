package browser

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func closeRows(version string, ids ...string) []sessionRow {
	rows := make([]sessionRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, sessionRow{
			SessionID:        id,
			Extension:        extensionInfo{Connected: true, Version: version},
			SessionPageCount: 1,
		})
	}
	return rows
}

func closeWindow(id int64, sessions ...string) windowSnapshot {
	return windowSnapshot{WindowID: id, Type: "normal", SessionIDs: sessions, Title: "Window"}
}

func closePayload(open ...windowSnapshot) string {
	features := []string{"all-windows", "window-close"}
	data, _ := json.Marshal(windowsPayload{
		Type:     "windows",
		Version:  "1.0.22",
		Features: features,
		Windows:  open,
	})
	return string(data)
}

func closeFake(rows []sessionRow, open ...windowSnapshot) *fakeClient {
	payload := closePayload(open...)
	f := &fakeClient{rows: rows}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		if call.Type == "windows" {
			return json.RawMessage(payload), true
		}
		return nil, false
	}
	return f
}

func closeFakeWithoutFeature(rows []sessionRow, open ...windowSnapshot) *fakeClient {
	data, _ := json.Marshal(windowsPayload{
		Type:     "windows",
		Version:  "1.0.21",
		Features: []string{"all-windows"},
		Windows:  open,
	})
	f := &fakeClient{rows: rows}
	f.handle = func(call fakeCall) (json.RawMessage, bool) {
		if call.Type == "windows" {
			return json.RawMessage(data), true
		}
		return nil, false
	}
	return f
}

func newCloseManager(t *testing.T, f *fakeClient) *Manager {
	t.Helper()
	m := newTestManager(t, f)
	m.sessionID = "s1"
	m.connected = true
	m.windowID = 900
	m.daemonVersion = "1.0.22"
	return m
}

func TestCloseWindowNonCurrentKeepsCurrent(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1", "s2"),
		closeWindow(900, "s1"), closeWindow(910, "s2"))
	m := newCloseManager(t, f)

	result, err := m.CloseWindow(context.Background(), 910, "")
	if err != nil {
		t.Fatalf("CloseWindow: %v", err)
	}
	if result.WindowID != 910 || result.SessionID != "s2" || result.Current != "" {
		t.Fatalf("result = %+v", result)
	}
	if m.sessionIDValue() != "s1" {
		t.Fatalf("current changed to %q", m.sessionIDValue())
	}
	call, ok := f.last("window")
	if !ok {
		t.Fatal("no window job recorded")
	}
	if call.Session != "s1" {
		t.Fatalf("close anchor = %q, want s1", call.Session)
	}
	if got := call.Params["op"]; got != "close" {
		t.Fatalf("op = %v", got)
	}
	if got := call.Params["window_id"]; got != int64(910) {
		t.Fatalf("window_id = %v", got)
	}
}

func TestCloseWindowSessionlessByID(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"), closeWindow(920))
	m := newCloseManager(t, f)

	result, err := m.CloseWindow(context.Background(), 0, sessionlessSessionID(920))
	if err != nil {
		t.Fatalf("CloseWindow: %v", err)
	}
	if result.WindowID != 920 || result.SessionID != sessionlessSessionID(920) {
		t.Fatalf("result = %+v", result)
	}
	if _, ok := f.last("window"); !ok {
		t.Fatal("close was not sent")
	}
}

func TestCloseWindowRefusesLastNormalWindow(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 900, ""); !errors.Is(err, errWindowCloseLast) {
		t.Fatalf("err = %v, want last window", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestCloseWindowCurrentNeedsLiveFallback(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"), closeWindow(920))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 0, "s1"); !errors.Is(err, errWindowCloseCurrent) {
		t.Fatalf("err = %v, want current fallback error", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestCloseWindowRefusesOldExtension(t *testing.T) {
	f := closeFakeWithoutFeature(closeRows("1.0.21", "s1", "s2"), closeWindow(900, "s1"), closeWindow(910, "s2"))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 910, ""); !errors.Is(err, errWindowCloseUnsupported) {
		t.Fatalf("err = %v, want unsupported", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestCloseWindowNeedsCloseCapableAnchor(t *testing.T) {
	f := closeFake(closeRows("1.0.21", "s1", "s2"), closeWindow(900, "s1"), closeWindow(910, "s2"))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 910, ""); !errors.Is(err, errWindowCloseUnsupported) && !errors.Is(err, errWindowCloseNoAnchor) {
		t.Fatalf("err = %v, want unsupported or no anchor", err)
	}
	if _, ok := f.last("window"); ok {
		t.Fatal("close job was sent")
	}
}

func TestCloseWindowRejectsBothSelectors(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"), closeWindow(910))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 910, "s2"); !errors.Is(err, errWindowCloseBoth) {
		t.Fatalf("err = %v, want both", err)
	}
}

func TestCloseWindowTargetGone(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"), closeWindow(910))
	m := newCloseManager(t, f)
	if _, err := m.CloseWindow(context.Background(), 999, ""); !errors.Is(err, errWindowGone) {
		t.Fatalf("err = %v, want gone", err)
	}
}

func TestCloseWindowReadOnly(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1"), closeWindow(900, "s1"), closeWindow(910))
	m := newCloseManager(t, f)
	m.readOnly = true
	if _, err := m.CloseWindow(context.Background(), 910, ""); !errors.Is(err, errReadOnly) {
		t.Fatalf("err = %v, want read only", err)
	}
}

func TestCloseWindowCurrentSwitchesToFallback(t *testing.T) {
	f := closeFake(closeRows("1.0.22", "s1", "s2"), closeWindow(900, "s1"), closeWindow(910, "s2"))
	f.info = `{"window_id":910,"tabs":[` +
		`{"id":911,"url":"http://127.0.0.1:43799/go?session=s2","active":true},` +
		`{"id":912,"url":"data:text/html,content","active":false}]}`
	f.window = `{"window_id":910,"window":{"left":0,"top":0,"width":1200,"height":900}}`
	m := newCloseManager(t, f)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := m.CloseWindow(ctx, 0, "s1")
	if err != nil {
		t.Fatalf("CloseWindow: %v", err)
	}
	if result.Current != "s2" {
		t.Fatalf("current = %q, want s2", result.Current)
	}
	if m.sessionIDValue() != "s2" {
		t.Fatalf("manager current = %q", m.sessionIDValue())
	}
	call, ok := f.last("window")
	if !ok {
		t.Fatal("no close job")
	}
	if call.Session != "s2" {
		t.Fatalf("close anchor = %q, want s2", call.Session)
	}
}
