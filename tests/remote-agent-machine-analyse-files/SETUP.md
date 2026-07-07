# Scenario

**Feature**: remote-agent machine analyse-files integration harness

```
# serverHome fixtures + server subprocess + isolated agent HOME
leaf Setup -> seed serverHome profile -> remote-agent machine analyse-files -> stdout blocks + summary
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` (global in each generated test) to scope a
   file cache under `$TMPDIR/machine-analyse-files-doctest-<session>/`
   (binaries built once per invocation).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaf packages.
3. Each leaf still gets an isolated `serverHome` / `agentHome`; only compiled binaries are shared.
4. Server runs with `HOME=serverHome` and cwd `serverHome` so scan scope matches fake machine home.
5. `git` is available in PATH for `git-dirs` profile seeding.

## Steps

1. Root `Run` builds binaries, seeds `serverHome` from `SeedProfile`, starts server, writes agent config.
2. Leaf `Setup` sets `SeedProfile` and optional `Args`.
3. `Run` executes `machine analyse-files` against live server.
4. Leaf `Assert` checks exit code, streamed entry blocks, and summary lines.

## Context

Implements REQUIREMENT-DESIGN-machine-analyse-files.md. Tests are expected to fail
until `machine analyse-files` and `server/machineanalyse` are implemented.

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
)

func sessionCacheDir() string {
	return filepath.Join(os.TempDir(), "machine-analyse-files-doctest-"+DOCTEST_SESSION_ID)
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

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func writeServerFile(t *testing.T, home, rel string, content []byte) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, content, 0644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func writeServerText(t *testing.T, home, rel, content string) {
	t.Helper()
	writeServerFile(t, home, rel, []byte(content))
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

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "branch", "-M", "main")
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "Initial commit")
}

func seedCodexTree(t *testing.T, home string) {
	t.Helper()
	writeServerText(t, home, ".codex/sessions/thread-a/rollout-001.jsonl", `{"type":"session"}`+"\n")
	writeServerText(t, home, ".codex/sessions/thread-b/rollout-002.jsonl", `{"type":"session"}`+"\n")
	writeServerText(t, home, ".codex/skills/my-skill/SKILL.md", "# skill\n")
	writeServerText(t, home, ".codex/cache/warm.dat", "cache-bytes\n")
	writeServerText(t, home, ".codex/rules/default.md", "# rule\n")
}

func seedGrokTree(t *testing.T, home string) {
	t.Helper()
	writeServerText(t, home, ".grok/sessions/s1/state.json", "{}\n")
	writeServerText(t, home, ".grok/projects/p1/README.md", "# p\n")
	writeServerText(t, home, ".grok/skills/s1/SKILL.md", "# s\n")
}

func seedPlainDir(t *testing.T, home string) {
	t.Helper()
	writeServerText(t, home, "plain-dir/sub/nested.txt", "nested\n")
}

func seedNodeModulesEntry(t *testing.T, home string) {
	t.Helper()
	writeServerText(t, home, "nm-entry/node_modules/pkg/index.js", "module.exports = 1;\n")
	writeServerText(t, home, "nm-entry/src/deep/node_modules/nested/index.js", "module.exports = 2;\n")
	writeServerText(t, home, "nm-entry/src/app.js", "console.log('app');\n")
}

func seedGitDirsEntries(t *testing.T, home string) {
	t.Helper()
	withGit := filepath.Join(home, "with-git")
	if err := os.MkdirAll(withGit, 0755); err != nil {
		t.Fatalf("mkdir with-git: %v", err)
	}
	gitInitRepo(t, withGit)
	seedPlainDir(t, home)
}

func seedEntryOrderEntries(t *testing.T, home string) {
	t.Helper()
	seedCodexTree(t, home)
	writeServerText(t, home, "aaa-first/alpha.txt", "a\n")
	writeServerText(t, home, "mmm-mid/middle.txt", "m\n")
	writeServerText(t, home, "notes.txt", "line one\nline two\n")
	writeServerText(t, home, "zzz-last/omega.txt", "z\n")
}

func seedFileLinesEntries(t *testing.T, home string) {
	t.Helper()
	notes, err := os.ReadFile("testdata/notes.txt")
	if err != nil {
		t.Fatalf("read testdata/notes.txt: %v", err)
	}
	binary, err := os.ReadFile("testdata/binary.dat")
	if err != nil {
		t.Fatalf("read testdata/binary.dat: %v", err)
	}
	writeServerFile(t, home, "notes.txt", notes)
	writeServerFile(t, home, "binary.dat", binary)
}

func seedBasicEntries(t *testing.T, home string) {
	t.Helper()
	seedPlainDir(t, home)
	writeServerText(t, home, "notes.txt", "alpha\nbeta\n")
}

func seedAnalyseServerHome(t *testing.T, home, profile string) error {
	t.Helper()
	switch profile {
	case "basic":
		seedBasicEntries(t, home)
	case "codex":
		seedCodexTree(t, home)
		seedPlainDir(t, home)
	case "file-lines":
		seedFileLinesEntries(t, home)
	case "git-dirs":
		seedGitDirsEntries(t, home)
	case "node-modules":
		seedNodeModulesEntry(t, home)
	case "entry-order":
		seedEntryOrderEntries(t, home)
	case "topic-absent":
		seedCodexTree(t, home)
		// Intentionally omit .grok.
	default:
		return fmt.Errorf("unknown SeedProfile %q", profile)
	}
	return nil
}

func combinedHasAll(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(combined, n) {
			t.Fatalf("output missing %q;\nhave:\n%s", n, combined)
		}
	}
}

func combinedHasNone(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(combined, n) {
			t.Fatalf("output unexpectedly contains %q;\nhave:\n%s", n, combined)
		}
	}
}

func extractEntryBlockOrder(t *testing.T, combined string) []string {
	t.Helper()
	lines := strings.Split(combined, "\n")
	var names []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "> ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(trim, "> "))
		if name == "" || strings.Contains(name, " ") {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		t.Fatalf("no top-level entry headers (> name) found in output:\n%s", combined)
	}
	return names
}

func assertSortedEntryNames(t *testing.T, names []string) {
	t.Helper()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("entry blocks not alphabetically sorted: %v", names)
		}
	}
}

func extractEntryBlock(t *testing.T, combined, entryName string) string {
	t.Helper()
	marker := "> " + entryName
	idx := strings.Index(combined, marker)
	if idx < 0 {
		t.Fatalf("missing entry block %q in output:\n%s", entryName, combined)
	}
	rest := combined[idx+len(marker):]
	next := strings.Index(rest, "\n> ")
	if next < 0 {
		next = strings.Index(rest, "\nanalyse-files summary")
	}
	if next < 0 {
		return rest
	}
	return rest[:next]
}

func assertChildBeforeSemantic(t *testing.T, block, childMarker, semanticMarker string) {
	t.Helper()
	childIdx := strings.Index(block, childMarker)
	semanticIdx := strings.Index(block, semanticMarker)
	if childIdx < 0 {
		t.Fatalf("block missing child marker %q:\n%s", childMarker, block)
	}
	if semanticIdx < 0 {
		t.Fatalf("block missing semantic marker %q:\n%s", semanticMarker, block)
	}
	if childIdx > semanticIdx {
		t.Fatalf("child %q should appear before semantic %q in block:\n%s", childMarker, semanticMarker, block)
	}
}
```