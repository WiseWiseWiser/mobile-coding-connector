package browser

import (
	"testing"
	"time"
)

// publish() owns the hub lock to decide whether a frame is a duplicate, so any
// helper it calls must not take the same lock again. Regression test for a
// self-deadlock that froze the whole /browser page on its first frame.
func TestPublishKeepsHubLockReentrant(t *testing.T) {
	h := newHub(&Manager{mode: ModeCast})
	h.mu.Lock()
	h.running = true
	h.gen = 1
	h.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.publish(1, []byte{0xff, 0xd8, 0xff, 0xd9}, 20*time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked: it called a hub helper that re-locks the hub mutex")
	}

	stats := h.stats()
	if stats.Frames != 1 {
		t.Fatalf("frames = %d, want 1", stats.Frames)
	}
	if stats.LastSize != 4 {
		t.Fatalf("last size = %d, want 4", stats.LastSize)
	}
}

// Shot mode trades frame rate for quiet: the keepalive must stay longer than
// cast's so a still page does not keep the extension busy.
func TestKeepaliveFollowsMode(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want time.Duration
	}{
		{ModeCast, castKeepalive},
		{ModeShot, shotKeepalive},
	} {
		h := newHub(&Manager{mode: tc.mode})
		h.mu.Lock()
		got := h.keepaliveLocked()
		h.mu.Unlock()
		if got != tc.want {
			t.Fatalf("%s keepalive = %s, want %s", tc.mode, got, tc.want)
		}
	}
}
