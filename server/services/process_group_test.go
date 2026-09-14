package services

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessGroupIDResolvesParentGroupForChild(t *testing.T) {
	cmd := exec.Command("bash", "-c", "sleep 300 & sleep 300")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer stopProcessGroup(cmd.Process.Pid)

	parentPID := cmd.Process.Pid
	childPID, err := firstChildPID(parentPID)
	if err != nil {
		t.Fatalf("firstChildPID() error = %v", err)
	}

	parentPGID, err := processGroupID(parentPID)
	if err != nil {
		t.Fatalf("processGroupID(parent) error = %v", err)
	}
	childPGID, err := processGroupID(childPID)
	if err != nil {
		t.Fatalf("processGroupID(child) error = %v", err)
	}
	if parentPGID != childPGID {
		t.Fatalf("parent PGID = %d, child PGID = %d, want same group", parentPGID, childPGID)
	}
	if childPGID == childPID {
		t.Fatalf("child PID %d equals its PGID; want non-leader listener case", childPID)
	}
}

func TestStopProcessGroupKillsNonLeaderListenerPID(t *testing.T) {
	cmd := exec.Command("bash", "-c", "sleep 300 & sleep 300")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	parentPID := cmd.Process.Pid

	childPID, err := firstChildPID(parentPID)
	if err != nil {
		_ = stopProcessGroup(parentPID)
		t.Fatalf("firstChildPID() error = %v", err)
	}

	if err := stopProcessGroup(childPID); err != nil {
		_ = stopProcessGroup(parentPID)
		t.Fatalf("stopProcessGroup(child) error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(childPID) {
			_ = stopProcessGroup(parentPID)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = stopProcessGroup(parentPID)
	t.Fatalf("child listener PID %d still alive after stopProcessGroup", childPID)
}

func firstChildPID(parentPID int) (int, error) {
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(parentPID)).Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0, exec.ErrNotFound
	}
	return strconv.Atoi(lines[0])
}

func TestParseProcStatState(t *testing.T) {
	cases := []struct {
		in    string
		state byte
		ok    bool
	}{
		{"123 (bash) S 1 123 123 0 -1", 'S', true},
		{"9 (a b) Z 1 9 9 0 -1", 'Z', true},
		{"42 (weird) name) R 1 42", 'R', true},
		{"broken", 0, false},
		{"1 () ", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseProcStatState([]byte(tc.in))
		if ok != tc.ok || got != tc.state {
			t.Fatalf("parseProcStatState(%q)=(%q,%v) want (%q,%v)", tc.in, got, ok, tc.state, tc.ok)
		}
	}
}

func TestProcessAliveOwnPID(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("current process should be alive")
	}
	if processAlive(0) || processAlive(-1) {
		t.Fatal("invalid pids must be dead")
	}
}

func TestProcessAliveTreatsZombieAsDead(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("zombie /proc state is Linux-specific")
	}
	// Parent holds a zombie child: fork+exit in child, sleep in parent.
	cmd := exec.Command("python3", "-c", "import os,time\npid=os.fork()\nif pid==0:\n os._exit(0)\ntime.sleep(60)")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	var zombiePID int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		out, err := exec.Command("bash", "-c",
			"ps -o pid=,stat= --ppid "+strconv.Itoa(cmd.Process.Pid)+" | awk '$2 ~ /Z/ {print $1; exit}'").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			if s != "" {
				zombiePID, _ = strconv.Atoi(s)
				if zombiePID > 0 {
					break
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if zombiePID <= 0 {
		t.Fatal("failed to create zombie child")
	}
	if !processZombie(zombiePID) {
		t.Fatalf("pid %d should be zombie", zombiePID)
	}
	if processAlive(zombiePID) {
		t.Fatalf("processAlive(%d) = true for zombie; want false", zombiePID)
	}
	if err := stopProcessGroup(zombiePID); err != nil {
		t.Fatalf("stopProcessGroup(zombie) error = %v", err)
	}
}

func TestCreateOrUpdateRestartOfMissingPIDStillSaves(t *testing.T) {
	dir := t.TempDir()
	m := NewManagerAt(dir)
	enabled := true
	def := ServiceDefinition{
		Name:       "restart-flake",
		Command:    "sleep 300",
		WorkingDir: dir,
		Enabled:    &enabled,
	}

	saved, err := m.createOrUpdate(def, false)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Stale PID that is not alive (same class of failure as a zombie for stop).
	m.mu.Lock()
	procDef := m.definitions[0]
	m.processes[saved.ID] = &serviceProcess{
		def:     procDef,
		pid:     1<<30 - 1,
		desired: true,
		status:  StatusRunning,
	}
	m.mu.Unlock()

	procDef.Command = "sleep 301"
	out, err := m.createOrUpdate(procDef, true)
	if err != nil {
		t.Fatalf("createOrUpdate with restart should not fail after save: %v", err)
	}
	if out.ID != saved.ID {
		t.Fatalf("id=%s want %s", out.ID, saved.ID)
	}
	if out.Command != "sleep 301" {
		t.Fatalf("command not saved: %q", out.Command)
	}
	_ = m.stop(saved.ID, true, true)
}