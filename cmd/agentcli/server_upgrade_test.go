package agentcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapUnameToGOARCH(t *testing.T) {
	cases := map[string]string{
		"x86_64":  "amd64",
		"amd64":   "amd64",
		"aarch64": "arm64",
		"arm64":   "arm64",
		"":        "",
	}
	for in, want := range cases {
		if got := mapUnameToGOARCH(in); got != want {
			t.Fatalf("mapUnameToGOARCH(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapUnameToGOOS(t *testing.T) {
	cases := map[string]string{
		"Ubuntu 22.04.4 LTS": "linux",
		"GNU/Linux":          "linux",
		"Darwin":             "darwin",
		"macOS":              "darwin",
		"":                   "",
	}
	for in, want := range cases {
		if got := mapUnameToGOOS(in); got != want {
			t.Fatalf("mapUnameToGOOS(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseGOOSArchFromBinaryName(t *testing.T) {
	goos, goarch, ok := parseGOOSArchFromBinaryName("ai-critic-server-linux-amd64-v16")
	if !ok || goos != "linux" || goarch != "amd64" {
		t.Fatalf("got %s/%s ok=%v", goos, goarch, ok)
	}
	goos, goarch, ok = parseGOOSArchFromBinaryName("ai-critic-server-linux-arm64")
	if !ok || goos != "linux" || goarch != "arm64" {
		t.Fatalf("got %s/%s ok=%v", goos, goarch, ok)
	}
	if _, _, ok := parseGOOSArchFromBinaryName("ai-critic-server-v3"); ok {
		t.Fatal("expected not ok for version-only name")
	}
}

func TestValidateAICriticSourceDir(t *testing.T) {
	dir := t.TempDir()
	if err := validateAICriticSourceDir(dir); err == nil {
		t.Fatal("expected error for empty dir")
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/xhd2015/ai-critic\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateAICriticSourceDir(dir); err == nil {
		t.Fatal("expected error for missing bundle script")
	}
	if err := os.MkdirAll(filepath.Join(dir, "script", "bundle", "for-linux"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := validateAICriticSourceDir(dir); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestResolveUpgradeSourceDirExplicit(t *testing.T) {
	dir := t.TempDir()
	got, notice, err := resolveUpgradeSourceDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(dir)
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
	if !strings.Contains(notice, "using source:") || !strings.Contains(notice, "--source-dir") {
		t.Fatalf("notice = %q", notice)
	}
}
