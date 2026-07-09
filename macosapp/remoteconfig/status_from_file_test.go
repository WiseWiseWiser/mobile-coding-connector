package remoteconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	got := DefaultConfigPath("/Users/demo")
	want := filepath.Join("/Users/demo", ".ai-critic", "remote-agent-config.json")
	if got != want {
		t.Fatalf("DefaultConfigPath = %q, want %q", got, want)
	}
}

func TestStatusFromFile_validConfig_notNotConfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "remote-agent-config.json")
	body := `{
  "default": "https://agent.example.com",
  "domains": [
    {"server": "https://agent.example.com", "token": "secret-token-value"}
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	line, server, resolved, err := StatusFromFile(path)
	if err != nil {
		t.Fatalf("StatusFromFile: %v", err)
	}
	if !resolved {
		t.Fatal("expected resolved=true for valid default+domain")
	}
	if server != "https://agent.example.com" {
		t.Fatalf("server = %q", server)
	}
	notConfigured := FormatStatus(StateNotConfigured, "")
	if line == notConfigured {
		t.Fatalf("status stuck on not_configured: %q", line)
	}
	want := "Connected to https://agent.example.com"
	if line != want {
		t.Fatalf("status = %q, want %q", line, want)
	}
	if strings.Contains(line, "secret-token-value") {
		t.Fatalf("status leaked token: %q", line)
	}
}

func TestStatusFromFile_missingFile_notConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	line, server, resolved, err := StatusFromFile(path)
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if resolved || server != "" {
		t.Fatalf("resolved=%v server=%q, want empty", resolved, server)
	}
	want := FormatStatus(StateNotConfigured, "")
	if line != want {
		t.Fatalf("status = %q, want %q", line, want)
	}
}

func TestStatusFromFile_cliShape_defaultAndDomains(t *testing.T) {
	// Shape produced by remote-agent config / successful remote-agent ping setup.
	dir := t.TempDir()
	path := DefaultConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "default": "https://agent-fast.example.xyz/",
  "domains": [
    {
      "server": "https://agent-fast.example.xyz",
      "token": "deadbeef"
    }
  ],
  "project_bindings": [
    {
      "server": "https://agent-fast.example.xyz",
      "remote_dir": "/root/proj",
      "local_path": "/Users/u/proj"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	line, server, resolved, err := StatusFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved {
		t.Fatal("CLI-shaped config must resolve")
	}
	if server != "https://agent-fast.example.xyz" {
		t.Fatalf("server = %q (want normalized no trailing slash)", server)
	}
	if strings.Contains(line, "Not configured") {
		t.Fatalf("must not show Not configured when remote-agent config exists: %q", line)
	}
	if line != "Connected to https://agent-fast.example.xyz" {
		t.Fatalf("line = %q", line)
	}
}

func TestSave_preservesProjectBindings_viaStatusPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "remote-agent-config.json")
	cfg := &Config{
		Default: "https://example.com",
		Domains: []Domain{{Server: "https://example.com", Token: "old"}},
		ProjectBindings: []ProjectBinding{{
			Server:    "https://example.com",
			RemoteDir: "/home/u/proj",
			LocalPath: "/Users/u/proj",
		}},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || loaded == nil {
		t.Fatalf("Load: %v %#v", err, loaded)
	}
	loaded.Domains[0].Token = "new-secret"
	if err := Save(path, loaded); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil || again == nil {
		t.Fatal(err)
	}
	if len(again.ProjectBindings) != 1 {
		t.Fatalf("bindings wiped: %#v", again.ProjectBindings)
	}
	if again.ProjectBindings[0].RemoteDir != "/home/u/proj" {
		t.Fatalf("binding: %#v", again.ProjectBindings[0])
	}
	if again.Domains[0].Token != "new-secret" {
		t.Fatalf("token not updated: %q", again.Domains[0].Token)
	}
	line, _, resolved, err := StatusFromFile(path)
	if err != nil || !resolved {
		t.Fatalf("after save: line=%q resolved=%v err=%v", line, resolved, err)
	}
	if strings.Contains(line, "new-secret") {
		t.Fatalf("token in status: %q", line)
	}
}
