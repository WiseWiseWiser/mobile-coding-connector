package procmon

import (
	"syscall"
	"testing"
)

func TestParseSignal(t *testing.T) {
	sig, label, err := parseSignal("")
	if err != nil || sig != syscall.SIGTERM || label != "TERM" {
		t.Fatalf("default: %v %v %v", sig, label, err)
	}
	sig, label, err = parseSignal("KILL")
	if err != nil || sig != syscall.SIGKILL || label != "KILL" {
		t.Fatalf("kill: %v %v %v", sig, label, err)
	}
	if _, _, err := parseSignal("HUP"); err == nil {
		t.Fatal("expected error for HUP")
	}
}

func TestKillProcessPIDOne(t *testing.T) {
	_, err := KillProcess(1, "TERM")
	if err == nil || killStatus(err) != 403 {
		t.Fatalf("got %v", err)
	}
}
