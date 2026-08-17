// Package e2etest runs the autonomous tty-watch attach gate.
package e2etest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

type Request struct {
	Phase string
}

type Response struct {
	Skipped bool
	Output  string
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
	if req.Phase != "tty-watch-journey" {
		return nil, fmt.Errorf("unknown phase %q", req.Phase)
	}
	if _, err := exec.LookPath("tty-watch"); err != nil {
		t.Skipf("tty-watch not on PATH: %v", err)
	}
	modRoot, err := findModuleRoot(d.DOCTEST_ROOT)
	if err != nil {
		return nil, err
	}
	script := filepath.Join(modRoot, "script", "verify-terminal-attach-e2e.sh")
	cmd := exec.Command(script)
	cmd.Dir = modRoot
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	resp := &Response{Output: string(out)}
	if err != nil {
		return resp, fmt.Errorf("verify-terminal-attach-e2e.sh: %w\n%s", err, out)
	}
	return resp, nil
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}
