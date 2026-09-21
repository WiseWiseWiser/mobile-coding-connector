package agentcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestGoHelp(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"go", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "mod-proxy") || !strings.Contains(s, "mod-proxy-relay") {
		t.Fatalf("go -h missing subcommands:\n%s", s)
	}
}

func TestGoModProxyHelp(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(RemoteProfile(), []string{"go", "mod-proxy", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"status", "start", "stop", "enable", "disable", "logs", "--now"} {
		if !strings.Contains(s, want) {
			t.Fatalf("mod-proxy -h missing %q:\n%s", want, s)
		}
	}
}

func TestGoModProxyRelayHelp(t *testing.T) {
	var out bytes.Buffer
	err := RunWithWriters(LocalProfile(), []string{"go", "mod-proxy-relay", "-h"}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"--upstream", "--port", "enable", "GOPROXY", "local-agent", "keep-alive", "GOPROXY usage"} {
		if !strings.Contains(s, want) {
			t.Fatalf("mod-proxy-relay -h missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "ssh --serve") {
		t.Fatalf("mod-proxy-relay -h still mentions ssh --serve:\n%s", s)
	}
}

func TestFormatRelayUsageRunning(t *testing.T) {
	got := formatRelayUsage("21001", true)
	for _, want := range []string{
		"Usage (GOPROXY via this relay):",
		"export GOPROXY=http://127.0.0.1:21001,https://proxy.golang.org,direct",
		"export GOINSECURE=127.0.0.1",
		"curl -s http://127.0.0.1:21001/github.com/!burnt!sushi/toml/@v/list",
		"go mod download github.com/google/uuid@v1.6.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("running usage missing %q:\n%s", want, got)
		}
	}
}

func TestFormatRelayUsageStopped(t *testing.T) {
	active = LocalProfile()
	got := formatRelayUsage("21001", false)
	if !strings.Contains(got, "Usage (start first: local-agent go mod-proxy-relay start)") {
		t.Fatalf("stopped header:\n%s", got)
	}
	if !strings.Contains(got, "export GOPROXY=http://127.0.0.1:21001,https://proxy.golang.org,direct") {
		t.Fatalf("stopped missing exports:\n%s", got)
	}
	if strings.Contains(got, "curl ") || strings.Contains(got, "go mod download") {
		t.Fatalf("stopped should not include smoke/fetch:\n%s", got)
	}
}

func TestFormatRelayUsageDefaultPort(t *testing.T) {
	got := formatRelayUsage("", true)
	if !strings.Contains(got, "http://127.0.0.1:21001,") {
		t.Fatalf("empty port should default to 21001:\n%s", got)
	}
}
