package agentcli

import (
	"strings"
	"testing"
)

func TestLooksLikeRemoteAbsPath(t *testing.T) {
	if !looksLikeRemoteAbsPath("/root/workspace/xgo") {
		t.Fatal("abs path")
	}
	if looksLikeRemoteAbsPath("xgo") {
		t.Fatal("name should not look abs")
	}
}

func TestAdhocModeValidationMessages(t *testing.T) {
	// Exercise help text contract: modes are documented.
	if !strings.Contains(projectPullLocalHelp, "--mode git-fetch|download") {
		t.Fatal("help missing --mode")
	}
	if !strings.Contains(projectPullLocalHelp, "--adhoc") {
		t.Fatal("help missing --adhoc")
	}
}
