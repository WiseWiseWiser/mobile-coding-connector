package services

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool { return &v }

// startTestService creates a service in an isolated manager and starts it.
func startTestService(t *testing.T, m *Manager, name string, command string) string {
	t.Helper()
	def := ServiceDefinition{Name: name, Command: command, Enabled: boolPtr(true)}
	status, err := m.CreateOrUpdate(def)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if _, err := m.Start(status.ID); err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	return status.ID
}

func servicePID(t *testing.T, m *Manager, id string) int {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if proc := m.processes[id]; proc != nil {
		return proc.pid
	}
	return 0
}

// freePort asks the kernel for an unused port, then releases it.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

// portServing reports whether something answers on a loopback port. A connect
// probe is the honest test here: on macOS a v4 bind can succeed while a v6
// wildcard listener already holds the port, so "can I bind?" misreports it.
func portServing(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func waitForServing(t *testing.T, port int, want bool, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if portServing(port) == want {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestShutdownReleasesServicePort pins the property whose loss caused the
// outage: a service holding a port must release it on shutdown. The kill path is
// correct today — this guards it against a shutdown that stops too early.
func TestShutdownReleasesServicePort(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	port := freePort(t)
	id := startTestService(t, m, "port-holder", fmt.Sprintf("python3 -m http.server %d", port))

	if !waitForServing(t, port, true, 10*time.Second) {
		t.Fatalf("service never served port %d", port)
	}

	m.Shutdown()

	if !waitForServing(t, port, false, 10*time.Second) {
		t.Fatalf("port %d still served after Shutdown; the process outlived cleanup", port)
	}
	if pid := servicePID(t, m, id); pid != 0 {
		t.Fatalf("manager still tracks pid %d after shutdown", pid)
	}
}

// TestForceStopAllKillsSurvivor covers the interrupted-cleanup path: when the
// cleanup budget expires, whatever is still running must be killed rather than
// left holding its port for the next instance to trip over.
func TestForceStopAllKillsSurvivor(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	port := freePort(t)
	// A shell that ignores SIGTERM, so only the SIGKILL escalation can end it.
	startTestService(t, m, "stubborn", fmt.Sprintf("trap '' TERM; python3 -m http.server %d", port))

	if !waitForServing(t, port, true, 10*time.Second) {
		t.Fatalf("service never served port %d", port)
	}

	killed := m.forceStopAll()
	if len(killed) != 1 {
		t.Fatalf("forceStopAll() killed %v, want 1 survivor", killed)
	}
	if !waitForServing(t, port, false, 10*time.Second) {
		t.Fatalf("port %d still served after forceStopAll", port)
	}
}

// TestReconcileOrphansKillsPreviousInstanceProcess reproduces the llm-proxy
// outage without restarting anything: a previous instance's process is still
// running and still holds its port, and the new manager must reclaim it before
// autostart so its own copy can bind.
func TestReconcileOrphansKillsPreviousInstanceProcess(t *testing.T) {
	dir := t.TempDir()
	port := freePort(t)
	command := fmt.Sprintf("python3 -m http.server %d", port)

	// A previous instance starts the service and then "crashes": its record
	// survives on disk while the process keeps running.
	prev := NewManagerAt(dir)
	id := startTestService(t, prev, "llm-proxy-like", command)
	orphanPID := servicePID(t, prev, id)
	if orphanPID == 0 {
		t.Fatal("previous instance did not record a pid")
	}
	if !waitForServing(t, port, true, 10*time.Second) {
		t.Fatalf("orphan never served port %d", port)
	}

	// Simulate the crash: drop in-memory state without stopping anything.
	prev.mu.Lock()
	prev.processes = map[string]*serviceProcess{}
	prev.mu.Unlock()

	next := NewManagerAt(dir)
	reclaimed := next.ReconcileOrphans()
	if len(reclaimed) != 1 {
		t.Fatalf("ReconcileOrphans() = %v, want one reclaimed process", reclaimed)
	}
	if processAlive(orphanPID) {
		t.Fatalf("orphan pid %d survived reconciliation", orphanPID)
	}
	if !waitForServing(t, port, false, 10*time.Second) {
		t.Fatalf("port %d still served after reconciliation", port)
	}

	// The replacement must now start cleanly on the reclaimed port.
	if _, err := next.Start(id); err != nil {
		t.Fatalf("start after reconciliation: %v", err)
	}
	if !waitForServing(t, port, true, 10*time.Second) {
		t.Fatalf("replacement service never served port %d", port)
	}
	next.Shutdown()
}

// TestReconcileOrphansIgnoresRecycledPID guards the destructive case: a stale
// record whose pid now belongs to an unrelated process must not be killed.
func TestReconcileOrphansIgnoresRecycledPID(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)

	// Record a live, unrelated process (this test binary's own pid) under a
	// command that does not match what it is actually running.
	m.saveRuntimeState(serviceRuntimeState{
		Version: runtimeStateVersion,
		Processes: map[string]serviceRuntimeRecord{
			"svc-recycled": {
				PID:     os.Getpid(),
				Command: "/nonexistent/binary --definitely-not-running",
			},
		},
	})

	if reclaimed := m.ReconcileOrphans(); len(reclaimed) != 0 {
		t.Fatalf("ReconcileOrphans() killed an unrelated process: %v", reclaimed)
	}
	if !processAlive(os.Getpid()) {
		t.Fatal("test process was killed by reconciliation")
	}
}

// TestRuntimeRecordClearedOnStop asserts a clean stop leaves no record behind,
// so a later boot has nothing to reclaim.
func TestRuntimeRecordClearedOnStop(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	id := startTestService(t, m, "recorded", "sleep 300")

	state, err := m.loadRuntimeState()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Processes[id]; !ok {
		t.Fatal("starting a service did not persist a runtime record")
	}

	if err := m.Stop(id); err != nil {
		t.Fatalf("stop: %v", err)
	}
	state, err = m.loadRuntimeState()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Processes[id]; ok {
		t.Fatal("a stopped service still has a runtime record; a later boot would try to reclaim it")
	}
}

// TestRuntimeStateFileIsBesideServicesJSON documents the storage location.
func TestRuntimeStateFileIsBesideServicesJSON(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	want := filepath.Join(dir, "services-runtime.json")
	if got := m.runtimeStateFile(); got != want {
		t.Fatalf("runtimeStateFile() = %q, want %q", got, want)
	}
	if !strings.HasSuffix(m.servicesFile(), "services.json") {
		t.Fatalf("unexpected services file %q", m.servicesFile())
	}
}
