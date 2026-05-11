package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveClientUsesSavedTokenForExplicitMatchingServer(t *testing.T) {
	writeTestConfig(t)

	cli, err := resolveClient("https://agent.example.com", "", false)
	if err != nil {
		t.Fatalf("resolveClient() error = %v", err)
	}
	if cli.Token != "saved-token" {
		t.Fatalf("client token = %q, want saved token", cli.Token)
	}
}

func TestResolveClientMatchesExplicitServerWithTrailingSlash(t *testing.T) {
	writeTestConfig(t)

	cli, err := resolveClient("https://agent.example.com/", "", false)
	if err != nil {
		t.Fatalf("resolveClient() error = %v", err)
	}
	if cli.Token != "saved-token" {
		t.Fatalf("client token = %q, want saved token", cli.Token)
	}
	if cli.Server != "https://agent.example.com" {
		t.Fatalf("client server = %q, want trimmed server", cli.Server)
	}
}

func TestResolveClientExplicitTokenOverridesSavedToken(t *testing.T) {
	writeTestConfig(t)

	cli, err := resolveClient("https://agent.example.com", "override-token", true)
	if err != nil {
		t.Fatalf("resolveClient() error = %v", err)
	}
	if cli.Token != "override-token" {
		t.Fatalf("client token = %q, want explicit token", cli.Token)
	}
}

func TestResolveClientUsesSavedTokenFromLegacyConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".ai-critic")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data := []byte(`{"server":"https://legacy.example.com","token":"legacy-token"}`)
	if err := os.WriteFile(filepath.Join(configDir, "remote-agent-config.json"), data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cli, err := resolveClient("https://legacy.example.com", "", false)
	if err != nil {
		t.Fatalf("resolveClient() error = %v", err)
	}
	if cli.Token != "legacy-token" {
		t.Fatalf("client token = %q, want legacy token", cli.Token)
	}
}

func TestHasGlobalFlagStopsAtCommand(t *testing.T) {
	args := []string{"exec", "--token", "command-token"}
	if hasGlobalFlag(args, "--token") {
		t.Fatalf("hasGlobalFlag() reported command argument as global token")
	}
}

func writeTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if err := saveConfig(&agentConfig{
		Default: "https://agent.example.com",
		Domains: []domainConfig{
			{Server: "https://agent.example.com", Token: "saved-token"},
		},
	}); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}
}
