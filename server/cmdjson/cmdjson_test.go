package cmdjson

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunParsesStdoutJSONAndRetainsStderrWarning(t *testing.T) {
	cmd := exec.Command("sh", "-c", `printf '{"ok":true}'; printf 'version is outdated\n' >&2`)
	type output struct {
		OK bool `json:"ok"`
	}

	result, err := Run[output](cmd)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Data.OK {
		t.Fatalf("decoded OK = false, want true")
	}
	if got := result.Warning(); got != "version is outdated" {
		t.Fatalf("warning = %q, want version warning", got)
	}
}

func TestRunErrorIncludesStderr(t *testing.T) {
	cmd := exec.Command("sh", "-c", `printf 'bad things\n' >&2; exit 7`)

	_, err := Run[struct{}](cmd)
	if err == nil {
		t.Fatalf("Run() error = nil, want command failure")
	}
	if !strings.Contains(err.Error(), "bad things") {
		t.Fatalf("Run() error = %q, want stderr included", err.Error())
	}
}
