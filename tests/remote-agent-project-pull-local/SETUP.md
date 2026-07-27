# Scenario

**Feature**: remote-agent project bind-local and pull-local harness (L2 mass + L3 smokes)

```
# L2: projects + projectpull RegisterAPI + agentcli.Run | L3 UseCLI: product binaries
leaf Setup -> git repos + projects.json -> bind-local|pull-local -> worktree / config / porcelain
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-project-pull-local-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. `git` is available for bare repos, clones, submodules, and porcelain checks.
4. **L2** (default): in-process mux with `projects.RegisterAPIForFile` +
   `projectpull.RegisterAPI` + `agentcli.Run` with testhooks home override (no Setenv).
5. **L3** (`UseCLI` smokes only): product binaries with isolated `AI_CRITIC_HOME` / agent `HOME`.
6. Server-side package limits and guards run in-process for L2 (same handlers as production).

## Steps

1. Root `Run` creates config/agent homes, writes `projects.json` and `remote-agent-config.json`.
2. Leaf `Setup` creates remote project repo (and local repo or plain dir); smokes set `UseCLI`.
3. L2 starts in-process APIs; L3 starts product server.
4. `Run` executes `bind-local` / `pull-local` via agentcli or product binary (optional piped stdin / two-phase).
5. Leaf `Assert` checks exit code, output, config bindings, worktree paths, and remote cleanliness.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-project-pull-local.md. L2 covers binding,
guards, and package transfer; two L3 smokes keep the product binary path.

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
	return filepath.Join(os.TempDir(), "remote-agent-project-pull-local-doctest-"+sessionID)
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

func gitAllowFileProtocolEnv() []string {
	env := append([]string{}, os.Environ()...)
	// Strip any prior GIT_CONFIG_* so we fully control protocol.file.allow for
	// file:// clones/submodules used by leaf Setup (modern git blocks file by default).
	out := make([]string, 0, len(env)+3)
	for _, e := range env {
		if strings.HasPrefix(e, "GIT_CONFIG_COUNT=") ||
			strings.HasPrefix(e, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(e, "GIT_CONFIG_VALUE_") {
			continue
		}
		out = append(out, e)
	}
	return append(out,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=protocol.file.allow",
		"GIT_CONFIG_VALUE_0=always",
	)
}

func gitArgsWithHookBypass(args []string) []string {
	// Global pre-commit hooks in some developer environments reject submodule
	// gitlinks; doctest fixtures must still commit them.
	if len(args) > 0 && args[0] == "commit" {
		out := make([]string, 0, len(args)+1)
		out = append(out, "commit", "--no-verify")
		out = append(out, args[1:]...)
		return out
	}
	return args
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	args = gitArgsWithHookBypass(args)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitAllowFileProtocolEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func gitRunC(t *testing.T, dir string, args ...string) {
	t.Helper()
	args = gitArgsWithHookBypass(args)
	full := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = gitAllowFileProtocolEnv()
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
	cmd.Env = gitAllowFileProtocolEnv()
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

// gitBranchShowCurrent runs git branch --show-current in dir.
func gitBranchShowCurrent(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --show-current in %s: %v\n%s", dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// gitSymbolicRefHEAD returns refs/heads/... for HEAD (fails if detached).
func gitSymbolicRefHEAD(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "symbolic-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git symbolic-ref HEAD in %s: %v\n%s", dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// assertWorktreeNamedBranch checks branch name matches worktree directory basename.
func assertWorktreeNamedBranch(t *testing.T, worktreePath string) {
	t.Helper()
	base := filepath.Base(worktreePath)
	branch := gitBranchShowCurrent(t, worktreePath)
	if branch == "" {
		t.Fatalf("worktree %s is detached (empty branch --show-current)", worktreePath)
	}
	if branch != base {
		t.Fatalf("branch %q want worktree basename %q", branch, base)
	}
	ref := gitSymbolicRefHEAD(t, worktreePath)
	wantRef := "refs/heads/" + branch
	if ref != wantRef {
		t.Fatalf("symbolic-ref %q want %q", ref, wantRef)
	}
}

// findWorktreeDirBySuffix returns a worktree path whose base name ends with suffix (e.g. main-1).
func findWorktreeDirBySuffix(t *testing.T, base, suffix string) string {
	t.Helper()
	var found string
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filepath.Base(path) == suffix {
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
				found = path
			}
		}
		return nil
	})
	if found == "" {
		t.Fatalf("no worktree dir %q under %s", suffix, base)
	}
	return found
}
```