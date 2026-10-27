package screen

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// stalledSource mimics a capture child process that starts fine but never
// emits a frame.
type stalledSource struct {
	closed chan struct{}
	once   sync.Once
}

func (s *stalledSource) next(ctx context.Context) ([]byte, error) {
	select {
	case <-s.closed:
		return nil, errors.New("capture command closed")
	case <-ctx.Done():
		return nil, errCaptureStopped
	}
}

func (s *stalledSource) close() { s.once.Do(func() { close(s.closed) }) }

func TestCaptureStartTimeoutFails(t *testing.T) {
	m, _, _ := testManager(t)
	m.mu.Lock()
	m.gen = 1
	m.state = StateStarting
	m.stopped = make(chan struct{})
	cfg := m.cfg
	m.mu.Unlock()

	oldTimeout, oldStart := captureStartTimeout, startSourceFunc
	captureStartTimeout = 40 * time.Millisecond
	startSourceFunc = func(ctx context.Context, maxWidth, fps, quality int) (source, error) {
		return &stalledSource{closed: make(chan struct{})}, nil
	}
	t.Cleanup(func() {
		captureStartTimeout = oldTimeout
		startSourceFunc = oldStart
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go m.captureLoop(ctx, 1, cfg, done)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("capture loop kept waiting on a source that never produced a frame")
	}

	st := m.Snapshot()
	if st.State != StateError {
		t.Fatalf("state = %q, want %q", st.State, StateError)
	}
	if !strings.Contains(st.Error, "no frames") {
		t.Fatalf("error = %q, want it to mention the missing frames", st.Error)
	}
}
