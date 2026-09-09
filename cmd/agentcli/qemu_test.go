package agentcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestQemuHelpLocal(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"qemu", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"local-agent qemu", "status", "cloudflared", "config", "~/.ai-critic/qemu"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}

func TestQemuHelpRemote(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"qemu", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "remote-agent qemu") {
		t.Fatalf("%s", out.String())
	}
}

func TestTopLevelHelpListsQemu(t *testing.T) {
	local := topLevelHelp(LocalProfile())
	remote := topLevelHelp(RemoteProfile())
	if !strings.Contains(local, "qemu") {
		t.Fatalf("local help missing qemu:\n%s", local)
	}
	if !strings.Contains(remote, "qemu") {
		t.Fatalf("remote help missing qemu:\n%s", remote)
	}
}

func TestQemuStartDryRunLocal(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"qemu", "start", "--dry-run"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "[dry-run] would") || !strings.Contains(s, "<guest_start.sh>") {
		t.Fatalf("%s", s)
	}
}

func TestQemuShowLocal(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"qemu", "show"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"ai-critic qemu", ".ai-critic/qemu", "22221"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in:\n%s", want, s)
		}
	}
}
