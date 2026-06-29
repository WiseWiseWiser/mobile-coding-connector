package skill

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	stdoutCh := make(chan []byte, 1)
	readErrCh := make(chan error, 1)
	go func() {
		data, readErr := io.ReadAll(reader)
		stdoutCh <- data
		readErrCh <- readErr
	}()

	os.Stdout = writer
	runErr := fn()
	os.Stdout = oldStdout
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	data := <-stdoutCh
	if err := <-readErrCh; err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return string(data), runErr
}

func TestSkillShowPrintsContent(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle([]string{"show"})
	})
	if err != nil {
		t.Fatalf("Handle(show): %v", err)
	}
	if !strings.Contains(stdout, "# Remote Agent Skill") {
		t.Fatalf("expected SKILL.md content, got: %s", stdout)
	}
}

func TestSkillInstallCodex(t *testing.T) {
	tmpDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(prevWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}

	err = Handle([]string{"install", "--codex"})
	if err != nil {
		t.Fatalf("Handle(install --codex): %v", err)
	}

	skillFile := filepath.Join(tmpDir, ".codex", "skills", "remote-agent", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if !strings.Contains(string(content), "# Remote Agent Skill") {
		t.Fatalf("unexpected skill content: %q", string(content))
	}
}

func TestSkillInstallCursor(t *testing.T) {
	tmpDir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(prevWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}

	err = Handle([]string{"install", "--cursor"})
	if err != nil {
		t.Fatalf("Handle(install --cursor): %v", err)
	}

	skillFile := filepath.Join(tmpDir, ".cursor", "skills", "remote-agent", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if !strings.Contains(string(content), "# Remote Agent Skill") {
		t.Fatalf("unexpected skill content: %q", string(content))
	}
}

func TestSkillInstallCustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "my-skill-dir")

	err := Handle([]string{"install", targetDir})
	if err != nil {
		t.Fatalf("Handle(install <dir>): %v", err)
	}

	skillFile := filepath.Join(targetDir, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if !strings.Contains(string(content), "# Remote Agent Skill") {
		t.Fatalf("unexpected skill content: %q", string(content))
	}
}

func TestSkillNoArgsShowsHelp(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle(nil)
	})
	if err != nil {
		t.Fatalf("Handle(nil): %v", err)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("expected help text, got: %s", stdout)
	}
}

func TestSkillUnknownSubcommandReturnsError(t *testing.T) {
	err := Handle([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "unknown skill command") {
		t.Fatalf("expected 'unknown skill command' error, got: %v", err)
	}
}
