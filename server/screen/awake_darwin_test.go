//go:build darwin

package screen

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// A stale frame is all a sleeping display gives AVFoundation, so the assertion
// has to live exactly as long as viewers do: never two children, and no child
// left behind once the last viewer is gone.
func TestDisplayAwakeFollowsViewerLifecycle(t *testing.T) {
	old := caffeinateCommand
	defer func() { caffeinateCommand = old }()
	started := 0
	caffeinateCommand = func() *exec.Cmd {
		started++
		return exec.Command("sleep", "30")
	}

	m, _, _ := testManager(t)
	m.startAwakeLocked()
	awake := m.awakeNow()
	if awake == nil || started != 1 {
		t.Fatalf("after start: awake=%v started=%d, want one child", awake, started)
	}
	proc := awake.Process
	m.startAwakeLocked()
	if started != 1 {
		t.Fatalf("second start spawned another child: started=%d", started)
	}

	m.stopAwakeLocked()
	if awake := m.awakeNow(); awake != nil {
		t.Fatalf("stop left awake=%v", awake)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		t.Fatalf("child %d still alive after stop", proc.Pid)
	}

	m.stopAwakeLocked()
	if started != 1 {
		t.Fatalf("stop respawned the child: started=%d", started)
	}
}

// awakeNow reads the field under the lock the waiter goroutine uses.
func (m *Manager) awakeNow() *exec.Cmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.awake
}
