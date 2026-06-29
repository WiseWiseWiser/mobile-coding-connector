package agentcli

import (
	"strings"
	"testing"
)

func TestRenderConfigHTMLLocalAgent(t *testing.T) {
	html, err := renderConfigHTML(LocalProfile())
	if err != nil {
		t.Fatalf("renderConfigHTML() error = %v", err)
	}
	for _, want := range []string{
		"Local Agent Config",
		"~/.ai-critic/local-agent-config.json",
		"local-agent upload",
		"--port",
		"http://localhost:23712",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered HTML missing %q\n%s", want, html)
		}
	}
	if strings.Contains(html, "remote-agent-config.json") {
		t.Fatalf("local config page must not mention remote-agent-config.json")
	}
}

func TestRenderConfigHTMLRemoteAgent(t *testing.T) {
	html, err := renderConfigHTML(RemoteProfile())
	if err != nil {
		t.Fatalf("renderConfigHTML() error = %v", err)
	}
	for _, want := range []string{
		"Remote Agent Config",
		"~/.ai-critic/remote-agent-config.json",
		"remote-agent upload",
		"https://host.example.com",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered HTML missing %q", want)
		}
	}
	if strings.Contains(html, "--port") {
		t.Fatalf("remote config page must not mention --port")
	}
}