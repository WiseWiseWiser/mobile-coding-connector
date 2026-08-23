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

func TestSkillShowRoot(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle([]string{"--show"})
	})
	if err != nil {
		t.Fatalf("Handle(--show): %v", err)
	}
	if !strings.Contains(stdout, "# remote-agent (CLI hub)") {
		t.Fatalf("expected hub SKILL.md, got: %s", stdout)
	}
	if !strings.Contains(stdout, "service upgrade") && !strings.Contains(stdout, "`service`") {
		t.Fatalf("expected topic index mentioning service, got: %s", stdout)
	}
}

func TestSkillShowAlias(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle([]string{"show"})
	})
	if err != nil {
		t.Fatalf("Handle(show): %v", err)
	}
	if !strings.Contains(stdout, "# remote-agent (CLI hub)") {
		t.Fatalf("expected hub content via alias, got: %s", stdout)
	}
}

func TestSkillShowTopic(t *testing.T) {
	for _, args := range [][]string{
		{"--show", "service"},
		{"service", "--show"},
	} {
		stdout, err := captureStdout(t, func() error {
			return Handle(args)
		})
		if err != nil {
			t.Fatalf("Handle(%v): %v", args, err)
		}
		if !strings.Contains(stdout, "service upgrade") {
			t.Fatalf("Handle(%v): expected service topic, got: %s", args, stdout)
		}
	}
}

func TestSkillListIncludesTopics(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle([]string{"--list"})
	})
	if err != nil {
		t.Fatalf("Handle(--list): %v", err)
	}
	for _, want := range []string{"remote-agent", "upload", "service", "seal", "config"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("--list missing %q; got:\n%s", want, stdout)
		}
	}
}

func TestSkillShowHeader(t *testing.T) {
	stdout, err := captureStdout(t, func() error {
		return Handle([]string{"--show", "--header"})
	})
	if err != nil {
		t.Fatalf("Handle(--show --header): %v", err)
	}
	if !strings.Contains(stdout, "name: remote-agent") {
		t.Fatalf("expected frontmatter name, got: %s", stdout)
	}
	if strings.Contains(stdout, "# remote-agent (CLI hub)") {
		t.Fatalf("header mode should not include body, got: %s", stdout)
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

	err = Handle([]string{"--install", "--codex"})
	if err != nil {
		t.Fatalf("Handle(--install --codex): %v", err)
	}

	skillFile := filepath.Join(tmpDir, ".codex", "skills", "remote-agent", "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if !strings.Contains(string(content), "# remote-agent (CLI hub)") {
		t.Fatalf("unexpected skill content: %q", string(content))
	}
	topicFile := filepath.Join(tmpDir, ".codex", "skills", "remote-agent", "service", "TOPIC.md")
	if _, err := os.Stat(topicFile); err != nil {
		t.Fatalf("expected nested topic after install: %v", err)
	}
}

func TestSkillInstallAliasCursor(t *testing.T) {
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
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("expected installed skill: %v", err)
	}
}

func TestSkillInstallCustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "my-skill-dir")

	err := Handle([]string{"--install", targetDir})
	if err != nil {
		t.Fatalf("Handle(--install <dir>): %v", err)
	}

	skillFile := filepath.Join(targetDir, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("read skill file: %v", err)
	}
	if !strings.Contains(string(content), "# remote-agent (CLI hub)") {
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
	if !strings.Contains(stdout, "Available topics:") && !strings.Contains(stdout, "upload") {
		t.Fatalf("expected topic index in help, got: %s", stdout)
	}
}

func TestSkillUnknownTopicReturnsError(t *testing.T) {
	err := Handle([]string{"--show", "bogus-topic"})
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
}
