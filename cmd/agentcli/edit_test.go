package agentcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/client"
)

func TestResolveEditorDefaultsToVim(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	editor, err := resolveEditor("")
	if err != nil {
		t.Fatal(err)
	}
	if editor.name != "vim" || len(editor.args) != 0 {
		t.Fatalf("editor = %+v, want vim with no args", editor)
	}
}

func TestResolveEditorPrecedence(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	t.Setenv("VISUAL", "emacs")

	// Flag beats $VISUAL; $VISUAL beats $EDITOR.
	editor, err := resolveEditor("micro")
	if err != nil {
		t.Fatal(err)
	}
	if editor.name != "micro" {
		t.Fatalf("flag editor = %q, want micro", editor.name)
	}

	editor, err = resolveEditor("")
	if err != nil {
		t.Fatal(err)
	}
	if editor.name != "emacs" {
		t.Fatalf("default editor = %q, want emacs ($VISUAL)", editor.name)
	}

	t.Setenv("VISUAL", "")
	editor, err = resolveEditor("")
	if err != nil {
		t.Fatal(err)
	}
	if editor.name != "nano" {
		t.Fatalf("default editor = %q, want nano ($EDITOR)", editor.name)
	}
}

func TestResolveEditorSplitsArguments(t *testing.T) {
	editor, err := resolveEditor("emacs -nw")
	if err != nil {
		t.Fatal(err)
	}
	if editor.name != "emacs" {
		t.Fatalf("name = %q", editor.name)
	}
	if got := strings.Join(editor.args, " "); got != "-nw" {
		t.Fatalf("args = %q, want -nw", got)
	}
}

func TestResolveEditorAddsWaitFlagForGUIEditors(t *testing.T) {
	tests := []struct {
		in       string
		wantName string
		wantArgs string
	}{
		{"code", "code", "--wait"},
		{"code-insiders", "code-insiders", "--wait"},
		{"codium", "codium", "--wait"},
		{"subl", "subl", "-w"},
		{"mate", "mate", "-w"},
		{"/usr/local/bin/code", "/usr/local/bin/code", "--wait"},
		{"code --wait", "code", "--wait"},
		{"code -n --wait", "code", "-n --wait"},
		{"vim", "vim", ""},
	}

	for _, tc := range tests {
		editor, err := resolveEditor(tc.in)
		if err != nil {
			t.Fatalf("resolveEditor(%q) error = %v", tc.in, err)
		}
		if editor.name != tc.wantName {
			t.Errorf("resolveEditor(%q) name = %q, want %q", tc.in, editor.name, tc.wantName)
		}
		if got := strings.Join(editor.args, " "); got != tc.wantArgs {
			t.Errorf("resolveEditor(%q) args = %q, want %q", tc.in, got, tc.wantArgs)
		}
	}
}

func TestExtractEditFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"equals form", []string{"file.txt", "--editor=code"}, "--editor=code"},
		{"space form", []string{"--editor", "code", "file.txt"}, "--editor code"},
		{"work dir", []string{"--work-dir=/tmp/x", "file.txt"}, "--work-dir=/tmp/x"},
		{"both", []string{"--editor=code", "--work-dir", "/tmp/x"}, "--editor=code --work-dir /tmp/x"},
		{"remember flag dropped", []string{"--editor=code", "--remember-flags"}, "--editor=code"},
		{"none", []string{"--remember-flags", "file.txt"}, ""},
		{"unknown flag ignored", []string{"--dry-run", "file.txt"}, ""},
	}
	for _, tc := range tests {
		got := strings.Join(extractEditFlags(tc.args), " ")
		if got != tc.want {
			t.Errorf("%s: extractEditFlags(%v) = %q, want %q", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestRememberedEditFlagsRoundTripAndPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	active = RemoteProfile()
	t.Cleanup(func() { active = Profile{} })

	// Nothing remembered yet.
	remembered, err := loadRememberedEditFlags()
	if err != nil {
		t.Fatal(err)
	}
	if len(remembered) != 0 {
		t.Fatalf("remembered = %v, want none", remembered)
	}

	if err := saveRememberedEditFlags([]string{"--editor=code"}); err != nil {
		t.Fatal(err)
	}
	remembered, err = loadRememberedEditFlags()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(remembered, " ") != "--editor=code" {
		t.Fatalf("remembered = %v", remembered)
	}

	// A run without flags applies the remembered editor.
	opts, applied, rest, err := parseEditOptions([]string{"file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.editor != "code" {
		t.Fatalf("editor = %q, want remembered code", opts.editor)
	}
	if strings.Join(applied, " ") != "--editor=code" {
		t.Fatalf("applied = %v", applied)
	}
	if len(rest) != 1 || rest[0] != "file.txt" {
		t.Fatalf("rest = %v", rest)
	}

	// An explicit flag wins over the remembered one.
	opts, _, _, err = parseEditOptions([]string{"--editor=nano", "file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.editor != "nano" {
		t.Fatalf("editor = %q, want explicit nano", opts.editor)
	}

	// Cleared memory stops applying anything.
	if err := saveRememberedEditFlags(nil); err != nil {
		t.Fatal(err)
	}
	opts, applied, _, err = parseEditOptions([]string{"file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.editor != "" || len(applied) != 0 {
		t.Fatalf("editor = %q applied = %v, want empty", opts.editor, applied)
	}

	// The stored config shape stays stable.
	data, err := os.ReadFile(filepath.Join(home, ".ai-critic", "remote-agent-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "remembered_flags") {
		t.Fatalf("cleared memory should be omitted from config:\n%s", data)
	}
}

func TestParseEditOptionsIgnoresUnknownRememberedFlags(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	active = RemoteProfile()
	t.Cleanup(func() { active = Profile{} })

	if err := saveRememberedEditFlags([]string{"--editor=code", "--gone-flag=1"}); err != nil {
		t.Fatal(err)
	}

	opts, applied, rest, err := parseEditOptions([]string{"file.txt"})
	if err != nil {
		t.Fatalf("unknown remembered flags must not fail the run: %v", err)
	}
	if opts.editor != "" || len(applied) != 0 {
		t.Fatalf("editor = %q applied = %v, want remembered flags dropped", opts.editor, applied)
	}
	if len(rest) != 1 || rest[0] != "file.txt" {
		t.Fatalf("rest = %v", rest)
	}
}

func TestFileMD5Hex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := fileMD5Hex(path)
	if err != nil {
		t.Fatal(err)
	}
	// md5 of zero bytes: the value `edit` sends for a new remote file.
	if got != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("md5(empty) = %q", got)
	}
	if _, err := fileMD5Hex(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEditConflictErrorMentionsBothSides(t *testing.T) {
	active = RemoteProfile()
	t.Cleanup(func() { active = Profile{} })

	err := editConflictError("/etc/app/config.yaml", "/tmp/remote-agent-edit/etc/app/config.yaml", &client.FileConflictError{
		Path:          "/etc/app/config.yaml",
		ExpectedMD5:   "base-md5",
		CurrentMD5:    "other-md5",
		CurrentExists: true,
	})
	msg := err.Error()
	for _, want := range []string{
		"/etc/app/config.yaml changed on the server",
		"downloaded md5: base-md5",
		"current  md5: other-md5",
		"your edits are kept at: /tmp/remote-agent-edit/etc/app/config.yaml",
		"remote-agent download /etc/app/config.yaml /tmp/remote-agent-edit/etc/app/config.yaml.remote",
		"remote-agent upload /tmp/remote-agent-edit/etc/app/config.yaml /etc/app/config.yaml",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("conflict message missing %q:\n%s", want, msg)
		}
	}
}

func TestEditHelpListsOptions(t *testing.T) {
	help := editHelpFor(RemoteProfile())
	for _, want := range []string{
		"remote-agent edit <REMOTE_PATH>",
		"--editor CMD",
		"--remember-flags",
		"--work-dir DIR",
		defaultEditWorkDir,
	} {
		if !strings.Contains(help, want) {
			t.Errorf("help missing %q", want)
		}
	}

	local := editHelpFor(LocalProfile())
	if !strings.Contains(local, "local-agent edit <REMOTE_PATH>") {
		t.Error("local help should use the local-agent name")
	}
}

// TestSaveConfigMergingDomainsPreservesUnknownFields guards the config web UI:
// it posts only {default, domains}, which used to replace the whole file and
// silently drop project bindings and remembered flags.
func TestSaveConfigMergingDomainsPreservesUnknownFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	active = RemoteProfile()
	t.Cleanup(func() { active = Profile{} })

	stored := &agentConfig{
		Default:         "http://a.example.com",
		Domains:         []domainConfig{{Server: "http://a.example.com", Token: "ta"}},
		ProjectBindings: []projectBinding{{Server: "http://a.example.com", RemoteDir: "/remote", LocalPath: "/local"}},
		RememberedFlags: map[string][]string{"edit": {"--editor=code"}},
	}
	if err := saveConfig(stored); err != nil {
		t.Fatal(err)
	}

	// Simulate the web UI: only default + domains.
	partial := &agentConfig{
		Default: "http://b.example.com",
		Domains: []domainConfig{{Server: "http://b.example.com", Token: "tb"}},
	}
	if err := saveConfigMergingDomains(partial); err != nil {
		t.Fatal(err)
	}

	got, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != "http://b.example.com" {
		t.Fatalf("default = %q", got.Default)
	}
	if len(got.Domains) != 1 || got.Domains[0].Token != "tb" {
		t.Fatalf("domains = %+v", got.Domains)
	}
	if len(got.ProjectBindings) != 1 || got.ProjectBindings[0].LocalPath != "/local" {
		t.Fatalf("project bindings were dropped: %+v", got.ProjectBindings)
	}
	if strings.Join(got.RememberedFlags["edit"], " ") != "--editor=code" {
		t.Fatalf("remembered flags were dropped: %+v", got.RememberedFlags)
	}

	data, err := os.ReadFile(filepath.Join(home, ".ai-critic", "remote-agent-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["project_bindings"]; !ok {
		t.Fatalf("project_bindings missing from saved config:\n%s", data)
	}
}
