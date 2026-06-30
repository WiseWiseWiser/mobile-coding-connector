# Scenario

**Feature**: remote-agent `project list` git status integration harness

```
# build server + remote-agent, seed projects.json, run project list
leaf Setup -> temp git repo -> projects.json -> remote-agent project list -> stdout
```

## Preconditions

1. Module builds `ai-critic-server` (`.`) and `remote-agent` (`./cmd/remote-agent`).
2. `git` is available in PATH for leaf setup steps.
3. Each test uses isolated `AI_CRITIC_HOME` with `lib.TestPassword` credentials.

## Steps

1. Root `Run` builds binaries and starts `ai-critic-server` on an ephemeral port.
2. Leaf `Setup` creates a temp project directory (git repo or plain dir) and fills `Request.Project`.
3. Root `Run` writes `projects.json` pointing at the absolute project `dir`.
4. Runs `remote-agent --server http://localhost:PORT --token testpassword project list`.
5. Captures stdout, stderr, and exit code for leaf assertions.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-project-list-git-status.md and
Local Dir bindings in `printProjectGitConfig`. Proves end-to-end wiring from
server `getGitStatus` through API to CLI list/git-config output.

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
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