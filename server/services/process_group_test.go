package services

import (
	"os/exec"
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