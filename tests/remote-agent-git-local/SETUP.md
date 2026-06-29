# Scenario

**Feature**: remote-agent `git -C` local subcommand harness

```
# build server + remote-agent, run git -C <dir> <args>
leaf Setup -> temp repo (or plain dir) -> remote-agent git ... -> stdout/stderr + exit
```

## Preconditions

1. Module builds `ai-critic-server` (`.`) and `remote-agent` (`./cmd/remote-agent`).
2. `git` is available in PATH for leaf setup steps.
3. Each test uses isolated `AI_CRITIC_HOME` with `lib.TestPassword` credentials.

## Steps

1. Root `Run` builds binaries and starts `ai-critic-server` on an ephemeral port.
2. Leaf `Setup` creates a temp directory (git repo or plain dir) and sets `Request.Args`
   and `Request.RepoDir`.
3. Runs `remote-agent --server http://localhost:PORT --token <test> <Args...>`.
4. Captures stdout, stderr, and exit code for leaf assertions.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-git-local-commands.md. Proves end-to-end
wiring for `POST /api/remote-agent/git/run` once implemented.

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func mkWorkDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-git-local-*")
	if err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func gitInitWithMain(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
}

func gitInitialCommit(t *testing.T, dir, message string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", message)
}

func gitSecondCommit(t *testing.T, dir, filename, message string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("second\n"), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	gitRun(t, dir, "add", filename)
	gitRun(t, dir, "commit", "-m", message)
}

func setGitLocalArgs(t *testing.T, req *Request, dir string, gitArgs ...string) {
	t.Helper()
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("abs dir: %v", err)
	}
	req.RepoDir = abs
	args := []string{"git", "-C", abs}
	args = append(args, gitArgs...)
	req.Args = args
}
```