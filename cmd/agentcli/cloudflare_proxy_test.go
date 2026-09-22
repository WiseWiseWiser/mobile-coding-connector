package agentcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCloudflareProxyHelpLocal(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"cloudflare-proxy", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"Usage: local-agent cloudflare-proxy",
		"status", "start", "stop", "enable", "disable",
		"add", "list", "delete",
		"127.0.0.1:23790",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("local help missing %q:\n%s", want, s)
		}
	}
}

func TestCloudflareProxyHelpRemote(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"cloudflare-proxy", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "Usage: remote-agent cloudflare-proxy") {
		t.Fatalf("remote help:\n%s", s)
	}
	if strings.Contains(s, "enable     auto-start") {
		t.Fatalf("remote help should not list daemon commands:\n%s", s)
	}
	for _, want := range []string{"add", "list", "delete", "--echo", "--forward"} {
		if !strings.Contains(s, want) {
			t.Fatalf("remote help missing %q:\n%s", want, s)
		}
	}
}

func TestCloudflareProxyRemoteStartRejected(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"cloudflare-proxy", "start"}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "local-agent-only") {
		t.Fatalf("err %v out %s", err, out.String())
	}
}

func TestCloudflareProxyAddRequiresHandler(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"cloudflare-proxy", "add", "--hostname", "foo.example.com"}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "--echo or --forward") {
		t.Fatalf("err %v out %s", err, out.String())
	}
}

func TestCloudflareProxyAddEchoAndForwardRejected(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{
		"cloudflare-proxy", "add",
		"--hostname", "foo.example.com",
		"--echo",
		"--forward", "http://127.0.0.1:23712",
	}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("err %v out %s", err, out.String())
	}
}

func TestCloudflareProxyAddHelp(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"cloudflare-proxy", "add", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "Usage: remote-agent cloudflare-proxy add") {
		t.Fatalf("add help:\n%s", s)
	}
}

func TestTopLevelHelpMentionsCloudflareProxy(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "cloudflare-proxy") {
		t.Fatalf("top help missing cloudflare-proxy:\n%s", out.String())
	}
}
