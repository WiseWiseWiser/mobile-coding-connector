# Scenario

**Feature**: remote-agent project bind-local and pull-local integration harness

```
# bare origin, remote + local clones, server + isolated HOME, run bind-local or pull-local
leaf Setup -> git repos + projects.json -> remote-agent -> worktree / config / remote porcelain
```

## Preconditions

1. Module builds `ai-critic-server` and `remote-agent`.
2. `git` is available for bare repos, clones, submodules, and porcelain checks.
3. Server uses isolated `AI_CRITIC_HOME`; agent uses isolated `HOME` for config and worktrees.

## Steps

1. Root `Run` builds binaries, starts server, writes `projects.json` and `remote-agent-config.json`.
2. Leaf `Setup` creates remote project repo (and local repo or plain dir) with shared or divergent origins.
3. `Run` executes `remote-agent` with optional piped stdin or two-phase pull for worktree collision.
4. Leaf `Assert` checks exit code, output, config bindings, worktree paths, and remote cleanliness.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-project-pull-local.md. Tests are expected to
fail until `bind-local` and `pull-local` are implemented in `cmd/agentcli/project.go`.

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

func mkProjectDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-remote-project-*")
	if err != nil {
		t.Fatalf("mkdir remote project: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func mkLocalDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-local-repo-*")
	if err != nil {
		t.Fatalf("mkdir local dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func mkBareDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ai-critic-bare-*")
	if err != nil {
		t.Fatalf("mkdir bare: %v", err)
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

func gitRunC(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %v: %v\n%s", dir, args, err, out)
	}
}

func gitInitWithMain(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
}

func seedBareOrigin(t *testing.T, bareDir string) string {
	t.Helper()
	gitRun(t, bareDir, "init", "--bare")
	seed := mkProjectDir(t)
	gitInitWithMain(t, seed)
	readme := filepath.Join(seed, "README.md")
	if err := os.WriteFile(readme, []byte("shared seed\n"), 0644); err != nil {
		t.Fatalf("write seed readme: %v", err)
	}
	gitRun(t, seed, "add", "README.md")
	gitRun(t, seed, "commit", "-m", "Initial commit")
	gitRun(t, seed, "remote", "add", "origin", bareDir)
	gitRun(t, seed, "push", "-u", "origin", "main")
	gitRunC(t, bareDir, "symbolic-ref", "HEAD", "refs/heads/main")
	absBare, err := filepath.Abs(bareDir)
	if err != nil {
		t.Fatalf("abs bare: %v", err)
	}
	return "file://" + absBare
}

func cloneFromOrigin(t *testing.T, dest, originURL string) {
	t.Helper()
	cmd := exec.Command("git", "clone", originURL, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git clone %s -> %s: %v\n%s", originURL, dest, err, out)
	}
	gitRunC(t, dest, "config", "user.email", "test@example.com")
	gitRunC(t, dest, "config", "user.name", "Test User")
	gitRunC(t, dest, "checkout", "main")
}

// RepoPair holds remote project dir and local clone paths.
type RepoPair struct {
	RemoteDir string
	LocalDir  string
}

// pairSameOriginRepos returns remote project dir and local repo sharing one bare origin.
func pairSameOriginRepos(t *testing.T) RepoPair {
	t.Helper()
	bare := mkBareDir(t)
	originURL := seedBareOrigin(t, bare)
	remoteDir := mkProjectDir(t)
	localDir := mkLocalDir(t)
	cloneFromOrigin(t, remoteDir, originURL)
	cloneFromOrigin(t, localDir, originURL)
	return RepoPair{RemoteDir: remoteDir, LocalDir: localDir}
}

// pairMismatchedOriginRepos returns repos whose origin URLs differ.
func pairMismatchedOriginRepos(t *testing.T) RepoPair {
	t.Helper()
	remoteBare := mkBareDir(t)
	localBare := mkBareDir(t)
	remoteOrigin := seedBareOrigin(t, remoteBare)
	localOrigin := seedBareOrigin(t, localBare)
	remoteDir := mkProjectDir(t)
	localDir := mkLocalDir(t)
	cloneFromOrigin(t, remoteDir, remoteOrigin)
	cloneFromOrigin(t, localDir, localOrigin)
	return RepoPair{RemoteDir: remoteDir, LocalDir: localDir}
}

func dirtyTopLevelModifiedAndUntracked(t *testing.T, dir string) {
	t.Helper()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("dirty remote\n"), 0644); err != nil {
		t.Fatalf("modify readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pulled-untracked.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatalf("write untracked: %v", err)
	}
}

func registerPullProject(t *testing.T, req *Request, id, name, remoteDir string) {
	t.Helper()
	req.Project = ProjectEntry{ID: id, Name: name, Dir: remoteDir}
}

func seedBindingForServer(t *testing.T, req *Request, remoteDir, localDir string) {
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
```