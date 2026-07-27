# Scenario

**Feature**: remote-agent `project list` git status harness (L2 mass + L3 smoke)

```
# L2: projects.RegisterAPIForFile + agentcli.Run | L3 UseCLI: product binaries
leaf Setup -> temp git repo -> projects.json -> project list -> stdout
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-project-list-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. `git` is available in PATH for leaf setup steps.
4. **L2** (default): in-process mux with `projects.RegisterAPIForFile` + bearer auth +
   `agentcli.Run` with testhooks home override (no process Setenv).
5. **L3** (`UseCLI` smoke only): product binaries with isolated `AI_CRITIC_HOME` / agent `HOME`.
6. Each test uses isolated config/agent homes with `lib.TestPassword` credentials.

## Steps

1. Root `Run` creates config/agent homes, writes projects.json and remote-agent-config.
2. Leaf `Setup` creates a temp project directory and fills `Request.Project`; smoke sets `UseCLI`.
3. L2 starts in-process projects API; L3 starts product server.
4. `Run` executes `project list` / `git-config get` via agentcli or product binary.
5. Leaf `Assert` checks exit code and CLI output.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-project-list-git-status.md and
Local Dir bindings in `printProjectGitConfig`. L2 covers git-status + binding
resolution; one L3 smoke keeps the product binary path.

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)


func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "remote-agent-project-list-doctest-"+sessionID)
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
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
	}
	if len(req.Args) == 0 {
		req.Args = []string{"project", "list"}
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

// mkProjectDir creates an isolated directory for a registered project.
func mkProjectDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-project-*")
	if err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// gitRun runs a git command in dir and fails the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// gitInitWithMain initialises a repo and normalises the default branch to main.
func gitInitWithMain(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
}

// gitInitialCommit adds README.md and commits with the given message.
func gitInitialCommit(t *testing.T, dir, message string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", message)
}

// mkLocalBindingDir is a temp path stored in project_bindings (need not be a git repo).
func mkLocalBindingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-local-binding-*")
	if err != nil {
		t.Fatalf("mkdir local binding dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// seedListBinding registers one binding row for the test server and remote project dir.
func seedListBinding(t *testing.T, req *Request, remoteDir, localDir string) {
	t.Helper()
	absRemote, err := filepath.Abs(remoteDir)
	if err != nil {
		t.Fatalf("abs remote: %v", err)
	}
	absLocal, err := filepath.Abs(localDir)
	if err != nil {
		t.Fatalf("abs local: %v", err)
	}
	req.SeedBindings = []ProjectBinding{{
		RemoteDir: absRemote,
		LocalPath: absLocal,
	}}
	req.LocalPath = absLocal
}

const localDirDashLine = "Local Dir:        -"

// assertLocalDirDash requires the dash placeholder when no binding matches.
func assertLocalDirDash(t *testing.T, stdout string) {
	t.Helper()
	if !strings.Contains(stdout, localDirDashLine) {
		t.Fatalf("stdout missing %q;\n%s", localDirDashLine, stdout)
	}
}
```