package unified_tunnel

import (
	"os/exec"
	"testing"
	"time"
)

func TestRestartDelayBackoff(t *testing.T) {
	got := []time.Duration{
		restartDelay(0),
		restartDelay(1),
		restartDelay(2),
		restartDelay(3),
		restartDelay(4),
	}
	want := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("restartDelay(%d) = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestUnexpectedExitRestartsOnce(t *testing.T) {
	utm, _ := testTunnelManager(t)
	var starts int
	cmd := &exec.Cmd{}
	t.Cleanup(SetTestProcessHooks(func(m *UnifiedTunnelManager) error {
		starts++
		m.running = true
		m.cmd = &exec.Cmd{}
		m.stopRequested = false
		return nil
	}, nil))
	utm.cmd = cmd
	utm.running = true
	utm.startedAt = time.Now()
	utm.sleepFor = func(time.Duration) {}

	utm.handleProcessExit(cmd)
	if starts != 1 {
		t.Fatalf("starts = %d, want 1", starts)
	}
	if !utm.running {
		t.Fatal("tunnel should be running after restart")
	}
}

func TestStopDoesNotRestart(t *testing.T) {
	utm, _ := testTunnelManager(t)
	var starts int
	cmd := &exec.Cmd{}
	t.Cleanup(SetTestProcessHooks(func(m *UnifiedTunnelManager) error {
		starts++
		return nil
	}, nil))
	utm.cmd = cmd
	utm.running = true
	utm.stopRequested = true
	utm.sleepFor = func(time.Duration) {}

	utm.handleProcessExit(cmd)
	if starts != 0 {
		t.Fatalf("starts = %d, want 0 after stop", starts)
	}
	if utm.running {
		t.Fatal("stopped tunnel should stay stopped")
	}
}

func TestRepeatedExitsUseBackoff(t *testing.T) {
	utm, _ := testTunnelManager(t)
	var delays []time.Duration
	utm.sleepFor = func(d time.Duration) { delays = append(delays, d) }
	t.Cleanup(SetTestProcessHooks(func(m *UnifiedTunnelManager) error {
		m.running = true
		m.cmd = &exec.Cmd{}
		m.stopRequested = false
		m.startedAt = time.Now()
		return nil
	}, nil))

	cmd := &exec.Cmd{}
	utm.cmd = cmd
	utm.running = true
	utm.startedAt = time.Now()
	for i := 0; i < 3; i++ {
		exited := utm.cmd
		utm.handleProcessExit(exited)
	}
	if len(delays) != 3 || delays[0] != 2*time.Second || delays[1] != 4*time.Second || delays[2] != 8*time.Second {
		t.Fatalf("delays = %v, want 2s, 4s, 8s", delays)
	}
}
