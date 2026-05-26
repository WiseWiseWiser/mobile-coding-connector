package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactURLSecretsMasksPasswordButKeepsHost(t *testing.T) {
	got := redactURLSecrets("http://alice:secret@example.com:3128")
	if got != "http://alice:%3Credacted%3E@example.com:3128" {
		t.Fatalf("redactURLSecrets() = %q", got)
	}
}

func TestRedactSSHCommandSecretsMasksIdentityFileAndProxyPassword(t *testing.T) {
	cmd := `"ssh" "-i" "/tmp/op-key-123" "-o" "ProxyCommand=curl http://alice:secret@example.com:3128"`
	got := redactSSHCommandSecrets(cmd)

	for _, want := range []string{
		`"<redacted-private-key-path>"`,
		`alice:%3Credacted%3E@example.com:3128`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("redactSSHCommandSecrets() missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "/tmp/op-key-123") || strings.Contains(got, "secret") {
		t.Fatalf("redactSSHCommandSecrets() leaked secret data in %q", got)
	}
}

func TestResolveCloneSourceExpandsRemoteHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveCloneSource("~/src/repo")
	if err != nil {
		t.Fatalf("resolveCloneSource() error = %v", err)
	}
	want := filepath.Join(home, "src", "repo")
	if got != want {
		t.Fatalf("resolveCloneSource() = %q, want %q", got, want)
	}
}

func TestResolveCloneSourceLeavesURLsUnchanged(t *testing.T) {
	for _, repo := range []string{
		"https://github.com/owner/repo.git",
		"git@example.com:owner/repo.git",
		"example.com:owner/repo.git",
	} {
		got, err := resolveCloneSource(repo)
		if err != nil {
			t.Fatalf("resolveCloneSource(%q) error = %v", repo, err)
		}
		if got != repo {
			t.Fatalf("resolveCloneSource(%q) = %q, want unchanged", repo, got)
		}
	}
}

func TestResolveCloneTargetDirDefaultsToRemoteHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveCloneTargetDir("/srv/src/repo.git", "")
	if err != nil {
		t.Fatalf("resolveCloneTargetDir() error = %v", err)
	}
	want := filepath.Join(home, "repo")
	if got != want {
		t.Fatalf("resolveCloneTargetDir() = %q, want %q", got, want)
	}
}

func TestAbsPathExpandsRemoteHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := absPath("~/dst")
	if err != nil {
		t.Fatalf("absPath() error = %v", err)
	}
	want := filepath.Join(home, "dst")
	if got != want {
		t.Fatalf("absPath() = %q, want %q", got, want)
	}
}

func TestWriteTokenAskPassCreatesTemporaryHelper(t *testing.T) {
	path, cleanup, err := writeTokenAskPass("secret-token")
	if err != nil {
		t.Fatalf("writeTokenAskPass() error = %v", err)
	}
	defer cleanup()
	if path == "" {
		t.Fatalf("writeTokenAskPass() path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read askpass helper: %v", err)
	}
	if strings.Contains(string(data), "secret-token") {
		t.Fatalf("askpass helper should not contain the token literal")
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("askpass helper still exists after cleanup: %v", err)
	}
}
