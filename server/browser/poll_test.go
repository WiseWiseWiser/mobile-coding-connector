package browser

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The poll fallback keeps a WebSocket-blocked viewer working over plain HTTP:
// /api/browser/frame serves the newest published JPEG (304 when the client is
// current), and /api/browser/input feeds the same input queue the websocket
// reader does.

func seedTestFrame(h *hub, jpeg []byte) uint32 {
	h.mu.Lock()
	h.seq++
	h.lastFrame = append([]byte(nil), jpeg...)
	seq := h.seq
	h.mu.Unlock()
	return seq
}

func TestFrameEndpointServesNewestAnd304(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.connected, m.contentTabID = "ai-critic", true, 902

	// No frame yet (the fake answers screenshot jobs with an empty payload, so
	// the loop cannot publish): the first poll waits its short grace, then 304s.
	w := httptest.NewRecorder()
	m.handleFrame(w, httptest.NewRequest(http.MethodGet, "/api/browser/frame", nil))
	if w.Code != http.StatusNotModified {
		t.Fatalf("empty cache HTTP %d, want 304", w.Code)
	}

	seq := seedTestFrame(m.hub, []byte{0xff, 0xd8, 0xff, 0xd9})

	// Without after: 200 with the bytes and the seq header.
	w = httptest.NewRecorder()
	m.handleFrame(w, httptest.NewRequest(http.MethodGet, "/api/browser/frame", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("frame HTTP %d, want 200", w.Code)
	}
	if got := w.Header().Get("X-Frame-Seq"); got != itoaFrame(seq) {
		t.Fatalf("X-Frame-Seq = %q, want %q", got, itoaFrame(seq))
	}
	if w.Body.String() != string([]byte{0xff, 0xd8, 0xff, 0xd9}) {
		t.Fatal("frame body is not the published jpeg")
	}

	// Up-to-date client: 304, zero bytes.
	w = httptest.NewRecorder()
	m.handleFrame(w, httptest.NewRequest(http.MethodGet, "/api/browser/frame?after="+itoaFrame(seq), nil))
	if w.Code != http.StatusNotModified {
		t.Fatalf("current client HTTP %d, want 304", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatal("304 carried a body")
	}

	// Stale client: 200 again (its after is behind the newest seq).
	w = httptest.NewRecorder()
	m.handleFrame(w, httptest.NewRequest(http.MethodGet, "/api/browser/frame?after=0", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("stale client HTTP %d, want 200", w.Code)
	}
}

func itoaFrame(n uint32) string {
	return strconv.FormatUint(uint64(n), 10)
}

// countDispatches counts cdp jobs whose CDP method matches, across the whole
// recorded call list.
func countDispatches(f *fakeClient, method string) int {
	n := 0
	for _, c := range f.all("cdp") {
		if m, _ := c.Params["method"].(string); m == method {
			n++
		}
	}
	return n
}

func TestFrameEndpointStartsAndKeepsCapture(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.connected, m.contentTabID = "ai-critic", true, 902

	m.handleFrame(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/browser/frame", nil))
	if !m.hub.running {
		t.Fatal("a poll request did not start capture")
	}
	if m.hub.stats().Viewers != 0 {
		t.Fatal("poll clients must not count as websocket viewers")
	}

	// The heartbeat ages out: the idle check schedules the usual stop grace.
	m.hub.mu.Lock()
	m.hub.lastPollAt = time.Now().Add(-2 * pollHeartbeat)
	m.hub.mu.Unlock()
	m.hub.pollIdle()
	if m.hub.stopTimer == nil {
		t.Fatal("aged-out poll did not schedule capture stop")
	}
}

func TestInputEndpointEnqueuesLikeWebSocket(t *testing.T) {
	f := healthyFake()
	m := newTestManager(t, f)
	m.sessionID, m.connected, m.contentTabID = "ai-critic", true, 902

	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		m.handleInput(w, httptest.NewRequest(http.MethodPost, "/api/browser/input", strings.NewReader(body)))
		return w
	}

	if w := post(`{"type":"tap","x":0.5,"y":0.25}`); w.Code != http.StatusOK {
		t.Fatalf("tap HTTP %d: %s", w.Code, w.Body.String())
	}
	// The input loop drains the queue and dispatches; a tap is press+release.
	deadline := time.Now().Add(2 * time.Second)
	for countDispatches(f, "Input.dispatchMouseEvent") < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("tap never dispatched; calls = %+v", f.all("cdp"))
		}
		time.Sleep(10 * time.Millisecond)
	}

	if w := post(`{"type":"key","key":"r","code":"KeyR","vk":82,"modifiers":4}`); w.Code != http.StatusOK {
		t.Fatalf("key HTTP %d: %s", w.Code, w.Body.String())
	}
	deadline = time.Now().Add(2 * time.Second)
	for countDispatches(f, "Input.dispatchKeyEvent") < 1 {
		if time.Now().After(deadline) {
			t.Fatalf("key never dispatched; calls = %+v", f.all("cdp"))
		}
		time.Sleep(10 * time.Millisecond)
	}

	if w := post(`{"type":"dance"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("unknown type HTTP %d, want 400", w.Code)
	}
	if w := post(`{"type":"tap","x":9,"y":0.5}`); w.Code != http.StatusBadRequest {
		t.Fatalf("unnormalized x HTTP %d, want 400", w.Code)
	}
}
