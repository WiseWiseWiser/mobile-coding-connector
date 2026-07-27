# Scenario

**Feature**: remote-agent `git -C` local subcommand harness (L2 mass + L3 smokes)

```
# L2: git.RegisterAPI + agentcli.Run | L3 UseCLI: product binaries
leaf Setup -> temp repo (or plain dir) -> git -C ... -> stdout/stderr + exit
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-git-local-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Module builds `ai-critic-server` (`.`) and `remote-agent` (`./cmd/remote-agent`) for L3 only.
4. `git` is available in PATH for leaf setup steps.
5. **L2** (default): in-process mux with `server/git.RegisterAPI` + `agentcli.Run`
   (no product binary `exec`; no process `HOME` mutation).
6. **L3** (`UseCLI` smokes only): product binaries with isolated `AI_CRITIC_HOME`.

## Steps

1. Root `Run` creates isolated config home; L2 starts in-process git API, L3 builds/starts product binaries.
2. Leaf `Setup` creates a temp directory (git repo or plain dir) and sets `Request.Args`
   and `Request.RepoDir`; smokes set `UseCLI`.
3. `Run` executes `git -C ...` via agentcli (L2) or product binary (L3).
4. Leaf `Assert` checks exit code and CLI output.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-git-local-commands.md. Proves wiring for
`POST /api/remote-agent/git/run` via L2 mass coverage and sparse L3 smokes.

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "remote-agent-git-local-doctest-"+sessionID)
}

func withFileLock(t *testing.T, lockPath string, fn func() error) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func buildSessionBinariesOnce(t *testing.T, moduleRoot, cacheDir string) (serverBin, agentBin string) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	serverBin = filepath.Join(cacheDir, "ai-critic-server")
	agentBin = filepath.Join(cacheDir, "remote-agent")
	ready := filepath.Join(cacheDir, "binaries.ready")
	lock := filepath.Join(cacheDir, "build.lock")
	err := withFileLock(t, lock, func() error {
		if fileExists(ready) && fileExists(serverBin) && fileExists(agentBin) {
			return nil
		}
		for _, spec := range []struct {
			out string
			pkg string
		}{
			{serverBin, "."},
			{agentBin, "./cmd/remote-agent"},
		} {
			cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
			cmd.Dir = moduleRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
			}
		}
		return os.WriteFile(ready, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("session binaries cache: %s", cacheDir)
	return serverBin, agentBin
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
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
