package screen

import (
	"encoding/json"
	"errors"
	"image"
	"testing"
	"time"
)

// Capture only costs something while somebody is watching: the host screen is
// recorded and the display is held awake. These cases pin the lifecycle that
// makes that true — capture starts on the first viewer, survives a short
// absence so a tab switch does not pay the cold start, and stops once the grace
// has passed. A hidden page counts as absent: it closes its socket.

// withStopGrace shortens the window so the tests do not sleep for ten seconds.
func withStopGrace(t *testing.T, grace time.Duration) {
	t.Helper()
	old := stopGrace
	stopGrace = grace
	t.Cleanup(func() { stopGrace = old })
}

// captureManager returns a manager that looks like a live capture with no
// viewers: state streaming, a cancel func and the stopped channel the real
// capture loop would close, and a primed tiler.
func captureManager(t *testing.T) *Manager {
	t.Helper()
	m, _, _ := testManager(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.viewers = make(map[*viewer]struct{})
	m.holder = nil
	m.holderSince = time.Time{}
	m.cancel = func() {}
	m.stopped = make(chan struct{})
	m.tiler = newTiler(TileSize, m.cfg.Quality)
	return m
}

func (m *Manager) captureRuntime() (running bool, state string, w, h int, timer bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel != nil, m.state, m.imgW, m.imgH, m.stopAt != nil
}

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

func TestLastViewerLeavingKeepsCaptureDuringGrace(t *testing.T) {
	withStopGrace(t, 150*time.Millisecond)
	m := captureManager(t)

	m.mu.Lock()
	m.scheduleStopLocked()
	m.mu.Unlock()

	// A tab switch is not the end of the session: the stream stays warm so the
	// next viewer joins without a visible gap.
	if running, _, _, _, timer := m.captureRuntime(); !running || !timer {
		t.Fatalf("running=%v timer=%v, want a pending stop and a live capture", running, timer)
	}
	waitFor(t, 3*time.Second, "stop after the grace", func() bool {
		running, _, _, _, _ := m.captureRuntime()
		return !running
	})
	running, state, w, h, timer := m.captureRuntime()
	if running {
		t.Fatal("capture is still running")
	}
	if state != StateStopped {
		t.Fatalf("state after the grace = %q, want %q", state, StateStopped)
	}
	if w != 0 || h != 0 {
		t.Fatalf("stream size after the stop = %dx%d, want 0x0", w, h)
	}
	if timer {
		t.Fatal("the stop timer was not cleared")
	}
}

func TestViewerReturningWithinGraceKeepsCapture(t *testing.T) {
	withStopGrace(t, 150*time.Millisecond)
	m := captureManager(t)

	m.mu.Lock()
	m.scheduleStopLocked()
	m.mu.Unlock()

	m.addViewer(nil)
	time.Sleep(300 * time.Millisecond)

	running, state, _, _, timer := m.captureRuntime()
	if !running {
		t.Fatal("capture stopped although a viewer came back inside the grace")
	}
	if state != StateStreaming {
		t.Fatalf("state = %q, want %q", state, StateStreaming)
	}
	if timer {
		t.Fatal("a viewer joining left the stop timer armed")
	}
}

func TestNoCaptureWithoutAViewer(t *testing.T) {
	m, _, _ := testManager(t)
	m.mu.Lock()
	m.viewers = make(map[*viewer]struct{})
	m.holder = nil
	m.cancel = nil
	m.stopped = nil
	m.state = StateStopped
	m.imgW, m.imgH = 0, 0
	m.mu.Unlock()

	// Everything a page can call before it opens the socket.
	got := m.Snapshot()

	if got.State != StateStopped || got.Viewers != 0 {
		t.Fatalf("status = %+v, want a stopped stream and no viewers", got)
	}
	running, _, w, _, timer := m.captureRuntime()
	if running {
		t.Fatal("a status read started capture")
	}
	if w != 0 {
		t.Fatalf("stream width = %d, want 0", w)
	}
	if timer {
		t.Fatal("a status read armed the stop timer")
	}
}

// A controller that disconnects — a hidden tab, a dropped tunnel — must not keep
// the lock, and must not cost the viewers who stayed a cold start. Releasing the
// lock injects releaseAll so no key stays down on the host.
func TestControllerDisconnectReleasesControl(t *testing.T) {
	m := captureManager(t)
	be := m.backend.(*fakeBackend)
	controller, other := newTestViewer("v-test"), newTestViewer("v-other")
	m.mu.Lock()
	m.viewers[controller] = struct{}{}
	m.viewers[other] = struct{}{}
	m.holder = controller
	m.holderSince = time.Now()
	m.mu.Unlock()

	m.removeViewer(controller)

	m.mu.Lock()
	holder := m.holder
	running, timer := m.cancel != nil, m.stopAt != nil
	viewers := len(m.viewers)
	m.mu.Unlock()

	if holder != nil {
		t.Fatalf("holder after the disconnect = %q, want nobody", holder.id)
	}
	if viewers != 1 {
		t.Fatalf("viewers = %d, want the one still watching", viewers)
	}
	if !containsCall(be.names(), "releaseAll") {
		t.Fatalf("backend.releaseAll not called: %v", be.names())
	}
	if !running || timer {
		t.Fatalf("running=%v timer=%v, want capture warm with no stop while a viewer remains", running, timer)
	}
	if got := m.Snapshot().Control.Holder; got != "" {
		t.Fatalf("status holder = %q, want empty", got)
	}

	// The viewer that stayed is told the lock is free without polling.
	var notice map[string]any
	for len(other.send) > 0 {
		msg := <-other.send
		if !msg.text {
			continue
		}
		var decoded map[string]any
		if json.Unmarshal(msg.data, &decoded) != nil {
			continue
		}
		if decoded["type"] == "control" {
			notice = decoded
		}
	}
	if notice == nil {
		t.Fatal("the remaining viewer was not told the control was released")
	}
	if notice["granted"] != false || notice["viewer_id"] != controller.id || notice["holder"] != "" {
		t.Fatalf("control notice = %v", notice)
	}
}

// The stop cancels the loop, but a frame that was already past accept() is
// still being decoded when it happens. That frame used to publish after the
// stop and flip the state back to streaming, leaving the manager reporting a
// capture with no child and no viewers until the next page opened — the status
// endpoint said "streaming" while nothing was being captured.
func TestLateCaptureResultsDoNotResurrectTheStream(t *testing.T) {
	m := captureManager(t)
	m.mu.Lock()
	m.gen = 7
	m.mu.Unlock()

	m.mu.Lock()
	m.stopCaptureLocked()
	m.mu.Unlock()

	// The in-flight frame lands after the stop.
	m.publish(7, image.NewRGBA(image.Rect(0, 0, 1280, 828)))
	if got := m.Snapshot(); got.State != StateStopped {
		t.Fatalf("state = %q after a late frame, want %q", got.State, StateStopped)
	}
	// So can the error that follows from killing the child.
	m.failCapture(7, errors.New("capture failed: EOF"))

	got := m.Snapshot()
	if got.State != StateStopped {
		t.Fatalf("state = %q after the stop, want %q", got.State, StateStopped)
	}
	if got.Capture.Width != 0 || got.Capture.Height != 0 {
		t.Fatalf("stream size = %dx%d, want 0x0", got.Capture.Width, got.Capture.Height)
	}
	if got.Capture.Published != 0 || got.Capture.Decoded != 0 {
		t.Fatalf("counters = %d/%d, want a stopped stream to keep them at zero", got.Capture.Published, got.Capture.Decoded)
	}
	if got.Error != "" {
		t.Fatalf("error = %q, want an empty status", got.Error)
	}
}

func containsCall(calls []string, want string) bool {
	for _, got := range calls {
		if got == want {
			return true
		}
	}
	return false
}
