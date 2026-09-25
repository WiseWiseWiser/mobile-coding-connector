package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// recordExec swaps the process-replacing exec for a recorder, restoring it on
// cleanup. Tests must never let a real syscall.Exec run.
func recordExec(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var calls []string
	orig := execProcess
	execProcess = func(bin string, argv []string, env []string) error {
		mu.Lock()
		calls = append(calls, bin)
		mu.Unlock()
		return errors.New("exec intercepted by test")
	}
	t.Cleanup(func() { execProcess = orig })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), calls...)
	}
}

func resetRestartState(t *testing.T) {
	t.Helper()
	prevMode := shutdownMode
	t.Cleanup(func() {
		shutdownMode = prevMode
		pendingRestartMu.Lock()
		pendingRestart = nil
		pendingRestartMu.Unlock()
	})
}

// TestRestartExecWaitsForCleanup is the regression test for the orphaned-service
// outage: the exec must not run until cleanup has finished. Previously the HTTP
// handler exec'd as soon as the shutdown was *initiated*, replacing the process
// image while services.Shutdown() was still stopping children — which left
// llm-proxy holding port 8890 on every restart.
func TestRestartExecWaitsForCleanup(t *testing.T) {
	execCalls := recordExec(t)
	resetRestartState(t)
	SetShutdownMode("restart")
	setPendingRestart("/tmp/next-binary")

	cleanupDone := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- awaitCleanupThenFinish(cleanupDone, 5*time.Second)
	}()

	// Cleanup is still running, so nothing may have exec'd yet.
	time.Sleep(150 * time.Millisecond)
	if got := execCalls(); len(got) != 0 {
		t.Fatalf("exec ran %d time(s) before cleanup finished", len(got))
	}

	close(cleanupDone)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("awaitCleanupThenFinish did not return after cleanup")
	}

	got := execCalls()
	if len(got) != 1 {
		t.Fatalf("exec ran %d time(s), want exactly 1", len(got))
	}
	if got[0] != "/tmp/next-binary" {
		t.Fatalf("exec ran %q, want the recorded binary", got[0])
	}
}

// TestRestartExecWaitsForHandlerRelease asserts the exec waits for the handler to
// finish writing its response, so the caller still receives the restart
// acknowledgement.
func TestRestartExecWaitsForHandlerRelease(t *testing.T) {
	execCalls := recordExec(t)
	resetRestartState(t)
	SetShutdownMode("restart")
	plan := setPendingRestart("/tmp/released-binary")

	cleanupDone := make(chan struct{})
	close(cleanupDone)

	finished := make(chan error, 1)
	go func() { finished <- awaitCleanupThenFinish(cleanupDone, time.Second) }()

	time.Sleep(150 * time.Millisecond)
	if got := execCalls(); len(got) != 0 {
		t.Fatal("exec ran before the handler released its response")
	}

	close(plan.released)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("awaitCleanupThenFinish did not return after release")
	}

	if got := execCalls(); len(got) != 1 {
		t.Fatalf("exec ran %d time(s), want exactly 1", len(got))
	}
}

// TestShutdownWithoutRestartDoesNotExec guards the plain shutdown path.
func TestShutdownWithoutRestartDoesNotExec(t *testing.T) {
	execCalls := recordExec(t)
	resetRestartState(t)
	shutdownMode = ""

	cleanupDone := make(chan struct{})
	close(cleanupDone)
	if err := awaitCleanupThenFinish(cleanupDone, time.Second); err != nil {
		t.Fatalf("awaitCleanupThenFinish() error = %v", err)
	}

	if got := execCalls(); len(got) != 0 {
		t.Fatalf("a plain shutdown exec'd %d time(s)", len(got))
	}
}

// TestRestartWithoutRecordedBinaryFails loudly instead of silently not
// restarting.
func TestRestartWithoutRecordedBinaryFails(t *testing.T) {
	execCalls := recordExec(t)
	resetRestartState(t)
	SetShutdownMode("restart")
	takePendingRestart() // clear anything left over

	cleanupDone := make(chan struct{})
	close(cleanupDone)
	err := awaitCleanupThenFinish(cleanupDone, time.Second)
	if err == nil {
		t.Fatal("restart with no recorded binary returned nil, want an error")
	}

	if got := execCalls(); len(got) != 0 {
		t.Fatal("exec ran despite no recorded binary")
	}
}

// TestExecRestartHandlerDoesNotExec is the direct guard on the original defect:
// the HTTP handler must record the restart and return, never replace the process
// itself.
func TestExecRestartHandlerDoesNotExec(t *testing.T) {
	execCalls := recordExec(t)
	resetRestartState(t)
	shutdownMu.Lock()
	prevRequested := shutdownRequested
	shutdownRequested = false
	shutdownMu.Unlock()
	t.Cleanup(func() {
		shutdownMu.Lock()
		shutdownRequested = prevRequested
		shutdownMu.Unlock()
	})

	req := httptest.NewRequest(http.MethodPost, "/api/server/exec-restart", nil)
	rec := httptest.NewRecorder()
	handleExecRestart(rec, req)

	if got := execCalls(); len(got) != 0 {
		t.Fatalf("the exec-restart handler exec'd %d time(s); it must leave that to the cleanup owner", len(got))
	}
	if takePendingRestart() == nil {
		t.Fatal("the handler did not record a pending restart for the cleanup owner to run")
	}
}
