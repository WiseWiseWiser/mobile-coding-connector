package agentcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCloudflaredHelp(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"cloudflared", "-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"Usage: remote-agent cloudflared",
		"status",
		"use native|qemu|proxy",
		"Exactly one of native, qemu, proxy",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestCloudflaredUseHelp(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"cloudflared", "use", "-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"Usage: remote-agent cloudflared use native|qemu|proxy",
		"native   host cloudflared",
		"qemu     cloudflared inside the qemu guest",
		"proxy    edge cloudflare-proxy",
		"proxy_url and token",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestCloudflaredStatusHelp(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"cloudflared", "status", "-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"Usage: remote-agent cloudflared status",
		"as soon as it is known",
		"backend is last",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestQemuStatusHelpStreams(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"qemu", "status", "-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "as soon as the probe returns") {
		t.Fatalf("help:\n%s", out.String())
	}
}

func TestQemuCloudflaredHelpMentionsStreaming(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"qemu", "cloudflared", "-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "lines appear as each probe returns") {
		t.Fatalf("help:\n%s", out.String())
	}
}

func TestCloudflaredUseRequiresBackend(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"cloudflared", "use"}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "native, qemu, or proxy") {
		t.Fatalf("err = %v", err)
	}
}

func TestCloudflaredUseRejectsUnknown(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"cloudflared", "use", "kvm"}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "native, qemu, or proxy") {
		t.Fatalf("err = %v", err)
	}
}

func TestTopLevelHelpMentionsCloudflared(t *testing.T) {
	var out bytes.Buffer
	if err := RunWithWriters(RemoteProfile(), []string{"-h"}, &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "cloudflared") {
		t.Fatalf("top help missing cloudflared:\n%s", out.String())
	}
}
