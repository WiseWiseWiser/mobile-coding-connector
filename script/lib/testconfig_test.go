package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTestConfigHome_underTempDir(t *testing.T) {
	configHome, err := CreateTestConfigHome()
	if err != nil {
		t.Fatalf("CreateTestConfigHome failed: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })

	if !IsUnderTempDir(configHome) {
		t.Fatalf("config home %q is not under temp dir %q", configHome, os.TempDir())
	}
}

func TestWriteTestCredentials_writesTestPassword(t *testing.T) {
	configHome, err := CreateTestConfigHome()
	if err != nil {
		t.Fatalf("CreateTestConfigHome failed: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(configHome) })

	credFile, err := WriteTestCredentials(configHome)
	if err != nil {
		t.Fatalf("WriteTestCredentials failed: %v", err)
	}

	data, err := os.ReadFile(credFile)
	if err != nil {
		t.Fatalf("read credentials: %v", err)
	}
	if string(data) != TestPassword+"\n" {
		t.Fatalf("credentials = %q, want %q", string(data), TestPassword+"\n")
	}
}

func TestQuickTestOptions_ensureConfigHome_isolatedFromHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	opts := &QuickTestOptions{}
	credFile, err := opts.ensureConfigHome()
	if err != nil {
		t.Fatalf("ensureConfigHome failed: %v", err)
	}
	t.Cleanup(func() { QuickTestCleanup(opts) })

	if !opts.managedConfigHome {
		t.Fatal("expected managed config home")
	}
	if !IsUnderTempDir(opts.ConfigHome) {
		t.Fatalf("config home %q is not under temp dir", opts.ConfigHome)
	}
	if strings.HasPrefix(opts.ConfigHome, filepath.Join(home, ".ai-critic")) {
		t.Fatalf("config home must not be under ~/.ai-critic, got %q", opts.ConfigHome)
	}
	if credFile != TestCredentialsFile(opts.ConfigHome) {
		t.Fatalf("credFile = %q, want %q", credFile, TestCredentialsFile(opts.ConfigHome))
	}
}

func TestAppendTestServerEnv_setsNoOpenBrowser(t *testing.T) {
	env := AppendTestServerEnv([]string{"FOO=bar", "AI_CRITIC_NO_OPEN_BROWSER=0"}, "/tmp/test-home")
	if !containsEnv(env, "AI_CRITIC_HOME=/tmp/test-home") {
		t.Fatalf("missing AI_CRITIC_HOME: %v", env)
	}
	if !containsEnv(env, "AI_CRITIC_NO_OPEN_BROWSER=1") {
		t.Fatalf("expected AI_CRITIC_NO_OPEN_BROWSER=1, got %v", env)
	}
}

func containsEnv(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}

func TestQuickTestOptions_ensureConfigHome_localSkipsTemp(t *testing.T) {
	opts := &QuickTestOptions{Local: true}
	credFile, err := opts.ensureConfigHome()
	if err != nil {
		t.Fatalf("ensureConfigHome failed: %v", err)
	}
	if credFile != "" || opts.ConfigHome != "" || opts.managedConfigHome {
		t.Fatalf("local mode should not create config home: credFile=%q configHome=%q managed=%v", credFile, opts.ConfigHome, opts.managedConfigHome)
	}
}