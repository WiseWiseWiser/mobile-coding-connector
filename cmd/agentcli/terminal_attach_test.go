package agentcli

import (
	"strings"
	"testing"

	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
)

func TestErrIfSessionNotAttachableExited(t *testing.T) {
	err := ErrIfSessionNotAttachable(ptyclient.SessionInfo{ID: "session-8", Status: "exited"})
	if err == nil {
		t.Fatal("expected error for exited session")
	}
	if !strings.Contains(err.Error(), "session-8 is exited") {
		t.Fatalf("got %q", err)
	}
}

func TestErrIfSessionNotAttachableRunning(t *testing.T) {
	if err := ErrIfSessionNotAttachable(ptyclient.SessionInfo{ID: "session-1", Status: "running"}); err != nil {
		t.Fatalf("running session must be attachable: %v", err)
	}
}

func TestTerminalAttachConnectOptionsUseAttachMode(t *testing.T) {
	opts := TerminalAttachConnectOptions("session-9")
	if opts.AttachMode != "attach" {
		t.Fatalf("AttachMode: got %q, want attach", opts.AttachMode)
	}
	if opts.AttachSnapshot {
		t.Fatal("AttachSnapshot must be off for in-place attach")
	}
}
