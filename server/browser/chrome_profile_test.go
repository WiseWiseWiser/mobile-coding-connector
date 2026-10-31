package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProfile(t *testing.T, root, name, extSettings string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Secure Preferences"), []byte(`{"extensions":{"settings":`+extSettings+`}}`), 0600); err != nil {
		t.Fatal(err)
	}
}

func makeAgentPackage(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "agent-package")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"name":"Browser Agent"}`), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDiscoverChromeProfileRequiresUnambiguousEnabledExtension(t *testing.T) {
	root := t.TempDir()
	pkg := makeAgentPackage(t, root)
	writeProfile(t, root, "Default", fmt.Sprintf(`{"a":{"path":%q,"state":1}}`, pkg))
	writeProfile(t, root, "Profile 1", fmt.Sprintf(`{"b":{"path":%q,"state":0}}`, pkg))
	got, err := discoverChromeProfile(root, "")
	if err != nil || got.Name != "Default" {
		t.Fatalf("profile=%+v err=%v", got, err)
	}
	writeProfile(t, root, "Profile 2", fmt.Sprintf(`{"c":{"path":%q,"disable_reasons":[]}}`, pkg))
	if _, err := discoverChromeProfile(root, ""); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("ambiguous err=%v", err)
	}
	got, err = discoverChromeProfile(root, "Profile 2")
	if err != nil || got.Name != "Profile 2" {
		t.Fatalf("explicit profile=%+v err=%v", got, err)
	}
	if _, err = discoverChromeProfile(root, "../Default"); err == nil {
		t.Fatal("path traversal accepted")
	}
}

func TestChromeProfileIgnoresDisabledAndMissingExtensions(t *testing.T) {
	root := t.TempDir()
	pkg := makeAgentPackage(t, root)
	writeProfile(t, root, "Default", fmt.Sprintf(`{"a":{"path":%q},"b":{"path":%q,"disable_reasons":[1]}}`, filepath.Join(root, "missing"), pkg))
	if _, err := discoverChromeProfile(root, ""); err == nil {
		t.Fatal("missing or disabled extensions accepted")
	}
}

func TestChromeProfileRejectsTwoEnabledAgentExtensions(t *testing.T) {
	root := t.TempDir()
	pkg := makeAgentPackage(t, root)
	writeProfile(t, root, "Default", fmt.Sprintf(`{"a":{"path":%q},"b":{"path":%q}}`, pkg, pkg))
	if _, err := discoverChromeProfile(root, ""); err == nil || !strings.Contains(err.Error(), "multiple enabled") {
		t.Fatalf("error=%v", err)
	}
}

func TestChromeProfileConflictingDisabledRecordFailsClosed(t *testing.T) {
	root := t.TempDir()
	pkg := makeAgentPackage(t, root)
	writeProfile(t, root, "Default", fmt.Sprintf(`{"a":{"path":%q}}`, pkg))
	file := filepath.Join(root, "Default", "Preferences")
	if err := os.WriteFile(file, []byte(`{"extensions":{"settings":{"a":{"state":0}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverChromeProfile(root, ""); err == nil {
		t.Fatal("disabled extension accepted")
	}
}

// Opt-in, read-only check. No cookies, history, profile files or processes are modified.
func TestCurrentChromeProfileReadOnly(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_CHROME_PROFILE") != "1" {
		t.Skip("host profile test disabled")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	p, err := discoverChromeProfile(filepath.Join(home, "Library/Application Support/Google/Chrome"), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("uniquely detected profile: %s", p.Name)
}
