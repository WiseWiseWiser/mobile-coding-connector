# Scenario

**Feature**: remote-agent machine backup and restore integration harness

```
# serverHome fixtures + server subprocess + isolated agent HOME
leaf Setup -> seed serverHome -> remote-agent machine backup|restore -> archive / stdout / server files
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` (global in each generated test) to scope a
   file cache under `$TMPDIR/machine-backup-doctest-<session>/`
   (binaries built once; default prereq archive built once when reuse applies).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaf packages.
3. Each leaf still gets an isolated `serverHome` / `agentHome`; only artifacts are shared.
4. Server runs with `HOME=serverHome` and cwd `serverHome` so backup scope matches fake machine home.

## Steps

1. Root `Run` builds binaries, seeds `serverHome`, starts server, writes agent config.
2. Leaf `Setup` narrows `Request` (flags, excludes, restore prereq backup, mutations).
3. `Run` may run a prereq `machine backup` before restore when `PrereqBackup` is set.
4. Leaf `Assert` checks exit code, CLI output, archive layout, and server home files.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-machine-backup.md,
REQUIREMENT-DESIGN-machine-backup-exclusions.md,
REQUIREMENT-DESIGN-excluded-sizes.md,
REQUIREMENT-DESIGN-backup-large-dir-summary.md,
REQUIREMENT-DESIGN-large-dir-detail-deep.md, and
REQUIREMENT-DESIGN-backup-config-refinements.md, and
REQUIREMENT-DESIGN-machine-backup-git-repos.md, and
REQUIREMENT-DESIGN-git-repos-home-scan.md. Git fixture leaves skip when `git`
is not on PATH (`requireGit`). `SeedGitReposNonDot` seeds `projects/demo` via
`seedGitReposNonDotFixture`. `SeedExcludedSizes` writes 1024 B /
512 B fixtures for per-rule EXCLUDED stats assertions. `SeedLargeDir` writes
`.big-test/` (>40 MB) and `.small-test/` for size-sorted DOT DIRS / LARGE SIZE
coverage. `SeedLargeDirDetailDeep` extends `SeedLargeDir` with `.deep-test/nested-big/`
(12 MB) and a small sibling; default `.cache` stays builtin-excluded. Flat
`LARGE DIR DETAIL` helpers parse `> <rel-path>  <size>` lines and assert
size-desc / path-asc sort. `SeedIncludedFetchSkills` writes small files under
paths removed from built-in exclusions.

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

func sessionCacheDir() string {
	return filepath.Join(os.TempDir(), "machine-backup-doctest-"+DOCTEST_SESSION_ID)
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

func needsCustomPrereqArchive(req *Request) bool {
	if len(req.ExcludePaths) > 0 || len(req.IncludePaths) > 0 {
		return true
	}
	return req.SeedDocker || req.SeedBackupMeta || req.SeedGitRepos || req.SeedGitReposWorktree || req.SeedGitReposEmpty
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func archiveHasXZMagicFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 6 {
		return false
	}
	magic := []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}
	return bytes.Equal(data[:6], magic)
}

const seededBackupMetaJSON = `{"seeded_meta":true,"marker":"pre-backup-old"}` + "\n"

type exclusionConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

type userBackupConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

type effectiveExclusionConfigJSON struct {
	Version           string `json:"version"`
	ExcludePaths      []struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
	} `json:"exclude_paths"`
	LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
}

func userBackupConfigPath(home string) string {
	return filepath.Join(home, ".ai-critic", "backup-config.json")
}

func parseUserBackupConfigJSON(t *testing.T, raw []byte) userBackupConfigJSON {
	t.Helper()
	var cfg userBackupConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse user backup-config.json: %v\n%s", err, raw)
	}
	return cfg
}

func parseEffectiveExclusionConfigJSON(t *testing.T, raw []byte) effectiveExclusionConfigJSON {
	t.Helper()
	var cfg effectiveExclusionConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse effective exclusion config: %v\n%s", err, raw)
	}
	return cfg
}

func assertPersistedExcludeEmptyReason(t *testing.T, raw []byte, path string) {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse persisted config map: %v\n%s", err, raw)
	}
	entriesRaw, ok := doc["exclude_paths"]
	if !ok {
		t.Fatal("persisted exclude_paths missing")
	}
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(entriesRaw, &entries); err != nil {
		t.Fatalf("parse persisted exclude_paths: %v\n%s", err, entriesRaw)
	}
	found := false
	for _, entry := range entries {
		pathRaw, ok := entry["path"]
		if !ok {
			continue
		}
		var gotPath string
		if err := json.Unmarshal(pathRaw, &gotPath); err != nil {
			t.Fatalf("parse persisted path: %v", err)
		}
		if gotPath != path {
			continue
		}
		found = true
		if reasonRaw, ok := entry["reason"]; ok {
			var reason string
			if err := json.Unmarshal(reasonRaw, &reason); err != nil {
				t.Fatalf("parse persisted reason for %q: %v", path, err)
			}
			if strings.TrimSpace(reason) != "" {
				t.Fatalf("persisted exclude_paths[%q].reason = %q, want empty or omitted", path, reason)
			}
		}
	}
	if !found {
		t.Fatalf("persisted exclude_paths missing %q; raw:\n%s", path, raw)
	}
	if strings.Contains(string(raw), `"user excluded"`) {
		t.Fatalf("persisted config must not contain user excluded reason; raw:\n%s", raw)
	}
}

func assertEffectiveExcludeReason(t *testing.T, cfg effectiveExclusionConfigJSON, path, wantReason string) {
	t.Helper()
	for _, e := range cfg.ExcludePaths {
		if e.Path == path {
			if e.Reason != wantReason {
				t.Fatalf("effective exclude_paths[%q].reason = %q, want %q", path, e.Reason, wantReason)
			}
			return
		}
	}
	t.Fatalf("effective exclude_paths missing %q: %+v", path, cfg.ExcludePaths)
}

func appendPostPrereqSetConfigExcludes(t *testing.T, home string, entries []PostSetConfigExcludeEntry) error {
	t.Helper()
	cfgPath := userBackupConfigPath(home)
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", cfgPath, err)
	}
	var doc struct {
		Version           string `json:"version"`
		ExcludePaths      []struct {
			Path   string `json:"path"`
			Reason string `json:"reason,omitempty"`
		} `json:"exclude_paths"`
		LargeDirThreshold string `json:"large_dir_threshold,omitempty"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", cfgPath, err)
	}
	existing := make(map[string]int, len(doc.ExcludePaths))
	for i, e := range doc.ExcludePaths {
		existing[e.Path] = i
	}
	for _, add := range entries {
		if idx, ok := existing[add.Path]; ok {
			doc.ExcludePaths[idx].Reason = add.Reason
			continue
		}
		doc.ExcludePaths = append(doc.ExcludePaths, struct {
			Path   string `json:"path"`
			Reason string `json:"reason,omitempty"`
		}{Path: add.Path, Reason: add.Reason})
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", cfgPath, err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(cfgPath, updated, 0644); err != nil {
		return fmt.Errorf("write %s: %w", cfgPath, err)
	}
	return nil
}

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func seedServerHome(t *testing.T, home string, withDocker bool) error {
	t.Helper()
	files := map[string]string{
		".bashrc":                    "export FAKE=1\n",
		".profile":                   "# profile\n",
		".ai-critic/ai-models.json":  `{"models":[]}` + "\n",
		".cargo/config.toml":         "[source.crates-io]\n",
		".cache/junk":                "cache data\n",
		".npm/x/package.json":        "{}\n",
		".cargo/registry/db/idx":     "registry\n",
		"Projects/visible.txt":       "not a dot entry\n",
	}
	for rel, content := range files {
		full := filepath.Join(home, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return err
		}
	}
	link := filepath.Join(home, ".local", "bin", "tool-link")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return err
	}
	if err := os.Symlink("../../.bashrc", link); err != nil {
		return err
	}
	if withDocker {
		dockerCfg := filepath.Join(home, ".docker", "config")
		if err := os.MkdirAll(filepath.Dir(dockerCfg), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dockerCfg, []byte(`{"auths":{}}`+"\n"), 0600); err != nil {
			return err
		}
	}
	seedExtendedExclusionFixtures(t, home)
	return nil
}

func elfStubBytes() []byte {
	// Minimal valid ELF64 LSB header (enough for magic-based detection).
	data := make([]byte, 104)
	copy(data, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0})
	data[18] = 0x3e // x86_64
	data[19] = 0x00
	return data
}

func sqliteStubBytes() []byte {
	data := make([]byte, 32)
	copy(data, []byte("SQLite format 3\x00"))
	return data
}

func jpegStubBytes() []byte {
	// Minimal JPEG SOI marker + padding (detected as image, not executable).
	data := make([]byte, 16)
	data[0] = 0xff
	data[1] = 0xd8
	data[2] = 0xff
	data[3] = 0xe0
	return data
}

func writeServerBinaryFile(t *testing.T, home, rel string, data []byte) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, data, 0755); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func seedExtendedExclusionFixtures(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".ai-critic/config.json", `{"fixture":true}`+"\n")
	writeServerFile(t, home, ".ai-critic/service.log", "service log line\n")
	writeServerFile(t, home, ".ai-critic/keep.log", "keep this log line\n")
	writeServerBinaryFile(t, home, ".ai-critic/bin/stub", elfStubBytes())
	writeServerFile(t, home, ".live-and-love/upload-chunks/chunk-1", "chunk data\n")
	writeServerBinaryFile(t, home, ".local/share/opencode/opencode.db", sqliteStubBytes())
	writeServerBinaryFile(t, home, ".live-and-love/imgs/photo.jpg", jpegStubBytes())
	writeServerFile(t, home, ".codex/.tmp/junk", "codex tmp\n")
	writeServerFile(t, home, ".local/share/opencode/repos/foo/clone", "repo clone\n")
	writeServerFile(t, home, ".local/share/opencode/log/app.log", "opencode app log\n")
	writeServerFile(t, home, ".local/share/cursor-agent/versions/v1/pkg", "agent version\n")
	writeServerBinaryFile(t, home, ".opencode/bin/opencode", elfStubBytes())
	writeServerFile(t, home, ".config/confluence-fetch-skill/data/cache", "confluence cache\n")
}

const (
	largeDirChildABytes           = 30 * 1024 * 1024
	largeDirChildBBytes           = 20 * 1024 * 1024
	deepNestedBigBytes            = 12 * 1024 * 1024
	deepSmallSiblingBytes         = 1024
	defaultLargeDirThresholdBytes = 10 * 1024 * 1024
	largeDirDetailThresholdBytes  = 10 * 1024 * 1024
)

func seedLargeDirFixture(t *testing.T, home string) {
	t.Helper()
	writeServerBinaryFile(t, home, ".big-test/child-a", paddedBytes(largeDirChildABytes))
	writeServerBinaryFile(t, home, ".big-test/child-b", paddedBytes(largeDirChildBBytes))
}

func seedSmallDirForSortFixture(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".small-test/tiny.txt", "small fixture\n")
}

func seedLargeDirDetailDeepFixture(t *testing.T, home string) {
	t.Helper()
	seedLargeDirFixture(t, home)
	writeServerBinaryFile(t, home, ".deep-test/nested-big/file", paddedBytes(deepNestedBigBytes))
	writeServerBinaryFile(t, home, ".deep-test/small/tiny", paddedBytes(deepSmallSiblingBytes))
}

func seedIncludedFetchSkills(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".config/git-fetch-skill/data/cache", "git-fetch cache\n")
	writeServerFile(t, home, ".config/confluence-fetch-skill/data/note", "confluence note\n")
	writeServerFile(t, home, ".knowledge-index/agents.json", `{"agents":[]}`+"\n")
}

func seedKnowledgeHub(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".knowledge-hub/cache/item", "knowledge cache\n")
	writeServerFile(t, home, ".knowledge-index/idx/data", "index data\n")
}

const (
	gitFixtureCommitMsg    = "backup git fixture"
	gitNonDotFixtureCommitMsg = "non-dot fixture"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	requireGit(t)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	requireGit(t)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
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

func gitWorktreeAdd(t *testing.T, mainDir, wtDir, branch string) {
	t.Helper()
	requireGit(t)
	if err := os.MkdirAll(filepath.Dir(wtDir), 0755); err != nil {
		t.Fatalf("mkdir worktree parent: %v", err)
	}
	gitRun(t, mainDir, "worktree", "add", "-b", branch, wtDir)
}

func seedGitReposNonDotFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	demoDir := filepath.Join(home, "projects", "demo")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", demoDir, err)
	}
	gitRun(t, demoDir, "init")
	gitRun(t, demoDir, "config", "user.email", "test@example.com")
	gitRun(t, demoDir, "config", "user.name", "Test User")
	gitRun(t, demoDir, "branch", "-M", "main")
	readme := filepath.Join(demoDir, "README.md")
	if err := os.WriteFile(readme, []byte("non-dot fixture\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, demoDir, "add", "README.md")
	gitRun(t, demoDir, "commit", "-m", gitNonDotFixtureCommitMsg)
}

func seedGitReposFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	mainDir := filepath.Join(home, ".wrk-test", "main")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", mainDir, err)
	}
	gitRun(t, mainDir, "init")
	gitRun(t, mainDir, "config", "user.email", "test@example.com")
	gitRun(t, mainDir, "config", "user.name", "Test User")
	gitRun(t, mainDir, "branch", "-M", "main")
	readme := filepath.Join(mainDir, "README.md")
	if err := os.WriteFile(readme, []byte("fixture\n"), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	gitRun(t, mainDir, "add", "README.md")
	gitRun(t, mainDir, "commit", "-m", gitFixtureCommitMsg)
}

func seedGitReposEmptyFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	emptyDir := filepath.Join(home, ".wrk-test", "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", emptyDir, err)
	}
	gitRun(t, emptyDir, "init")
	gitRun(t, emptyDir, "config", "user.email", "test@example.com")
	gitRun(t, emptyDir, "config", "user.name", "Test User")
	gitRun(t, emptyDir, "branch", "-M", "main")
}

func seedGitReposWorktreeFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	seedGitReposFixture(t, home)
	mainDir := filepath.Join(home, ".wrk-test", "main")
	wtDir := filepath.Join(home, ".wrk-test", "feature-wt")
	gitWorktreeAdd(t, mainDir, wtDir, "feature/foo")
	readme := filepath.Join(wtDir, "README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatalf("read worktree README.md: %v", err)
	}
	if err := os.WriteFile(readme, append(data, []byte("dirty line\n")...), 0644); err != nil {
		t.Fatalf("dirty worktree README.md: %v", err)
	}
}

func seedGitReposMaxDepthFixture(t *testing.T, home string) {
	t.Helper()
	requireGit(t)
	deepDir := filepath.Join(home, ".wrk-test", "a", "b", "c", "deep-repo")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", deepDir, err)
	}
	gitRun(t, deepDir, "init")
	gitRun(t, deepDir, "config", "user.email", "test@example.com")
	gitRun(t, deepDir, "config", "user.name", "Test User")
	gitRun(t, deepDir, "branch", "-M", "main")
	readme := filepath.Join(deepDir, "README.md")
	if err := os.WriteFile(readme, []byte("deep\n"), 0644); err != nil {
		t.Fatalf("write deep README.md: %v", err)
	}
	gitRun(t, deepDir, "add", "README.md")
	gitRun(t, deepDir, "commit", "-m", "deep fixture")
}

var gitShortSHARE = regexp.MustCompile(`\b[0-9a-f]{7}\b`)

func gitReposSummarySection(combined string) string {
	idxPlan := strings.Index(combined, "dry-run: machine backup plan")
	if idxPlan < 0 {
		return ""
	}
	rest := combined[idxPlan:]
	idxGit := strings.Index(rest, "GIT REPOS")
	if idxGit < 0 {
		return ""
	}
	rest = rest[idxGit:]
	idxTotal := strings.Index(rest, "  TOTAL:")
	if idxTotal < 0 {
		idxExcluded := strings.Index(rest, "\n  EXCLUDED")
		if idxExcluded >= 0 {
			return rest[:idxExcluded]
		}
		return rest
	}
	return rest[:idxTotal]
}

func assertGitReposSummaryContains(t *testing.T, combined string, needles ...string) {
	t.Helper()
	section := gitReposSummarySection(combined)
	if section == "" {
		t.Fatalf("missing GIT REPOS summary section; got:\n%s", combined)
	}
	for _, n := range needles {
		if !strings.Contains(section, n) {
			t.Fatalf("GIT REPOS section missing %q; section:\n%s", n, section)
		}
	}
}

func assertStdoutEndsWithNewline(t *testing.T, stdout string) {
	t.Helper()
	if stdout == "" {
		t.Fatal("stdout empty; want trailing newline")
	}
	if !strings.HasSuffix(stdout, "\n") {
		tail := 40
		if len(stdout) < tail {
			tail = len(stdout)
		}
		t.Fatalf("stdout missing trailing newline; ends with %q", stdout[len(stdout)-tail:])
	}
}

type gitRepoWorktreesSnapshot struct {
	Version    string `json:"version"`
	CapturedAt string `json:"captured_at"`
	Repos      []struct {
		Path      string `json:"path"`
		Branch    string `json:"branch"`
		CommitSHA string `json:"commit_sha"`
		CommitMsg string `json:"commit_msg"`
		Status    string `json:"status"`
		Worktrees []struct {
			Path      string `json:"path"`
			Branch    string `json:"branch"`
			CommitSHA string `json:"commit_sha"`
			CommitMsg string `json:"commit_msg"`
			Status    string `json:"status"`
		} `json:"worktrees,omitempty"`
	} `json:"repos"`
}

func parseGitRepoWorktreesJSON(t *testing.T, raw []byte) gitRepoWorktreesSnapshot {
	t.Helper()
	var snap gitRepoWorktreesSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse git-repo-worktrees.json: %v\n%s", err, raw)
	}
	return snap
}

func assertGitRepoSnapshotBasics(t *testing.T, snap gitRepoWorktreesSnapshot, repoPath string) {
	t.Helper()
	if snap.Version != "1.0" {
		t.Fatalf("git snapshot version = %q, want 1.0", snap.Version)
	}
	if snap.CapturedAt == "" {
		t.Fatal("git snapshot missing captured_at")
	}
	var found bool
	for _, repo := range snap.Repos {
		if repo.Path != repoPath {
			continue
		}
		found = true
		if len(repo.CommitSHA) != 7 || !gitShortSHARE.MatchString(repo.CommitSHA) {
			t.Fatalf("repo %q commit_sha = %q, want 7-char hex short sha", repoPath, repo.CommitSHA)
		}
		if repo.Status == "" {
			t.Fatalf("repo %q missing status", repoPath)
		}
	}
	if !found {
		t.Fatalf("git snapshot missing repo path %q; repos=%v", repoPath, snap.Repos)
	}
}

func exclusionConfigHasPath(cfg exclusionConfigJSON, path string) (reason string, ok bool) {
	for _, e := range cfg.ExcludePaths {
		if e.Path == path {
			return e.Reason, true
		}
	}
	return "", false
}

func assertExclusionConfigV11(t *testing.T, cfg exclusionConfigJSON) {
	t.Helper()
	if cfg.Version != "1.1" {
		t.Fatalf("version = %q, want 1.1", cfg.Version)
	}
	wantEntries := map[string]string{
		"**(binary)":                          "executable binaries (reinstallable)",
		"**/*.log":                            "log files",
		"**/upload-chunks":                    "incomplete upload temp state",
		".local/share/cursor-agent/versions":  "Cursor agent version cache",
		".opencode/bin":                       "OpenCode binary (reinstallable)",
		".codex/.tmp":                         "Codex temporary plugin cache",
		".codex/skills/.system":               "Codex system skills cache",
		".local/share/opencode/repos":         "OpenCode repo clone cache",
		".local/share/opencode/snapshot":      "OpenCode snapshot cache",
		".local/share/opencode/log":           "OpenCode application logs",
		".grok/marketplace-cache":             "Grok plugin marketplace git cache",
		".grok/vendor":                        "Grok vendored dependencies cache",
		".grok/logs":                          "Grok application logs",
		".cache":                              "temporary application cache",
		"**/node_modules":                     "node_modules directories",
	}
	for path, wantReason := range wantEntries {
		gotReason, ok := exclusionConfigHasPath(cfg, path)
		if !ok {
			t.Fatalf("exclude_paths missing %q", path)
		}
		if strings.TrimSpace(gotReason) == "" {
			t.Fatalf("exclude_paths[%q] missing reason", path)
		}
		if gotReason != wantReason {
			t.Fatalf("exclude_paths[%q].reason = %q, want %q", path, gotReason, wantReason)
		}
	}
}

func excludedSection(combined string) string {
	idx := strings.Index(combined, "EXCLUDED")
	if idx < 0 {
		return ""
	}
	return combined[idx:]
}

func dotFilesSection(combined string) string {
	idxFiles := strings.Index(combined, "DOT FILES")
	idxDirs := strings.Index(combined, "DOT DIRS")
	if idxFiles < 0 || idxDirs < 0 || idxFiles >= idxDirs {
		return ""
	}
	return combined[idxFiles:idxDirs]
}

func dotDirsSection(combined string) string {
	idxDirs := strings.Index(combined, "DOT DIRS")
	idxExcluded := strings.Index(combined, "EXCLUDED")
	if idxDirs < 0 || idxExcluded < 0 || idxDirs >= idxExcluded {
		return ""
	}
	return combined[idxDirs:idxExcluded]
}

func assertDotDirsExcludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotDirsSection(combined)
	if section == "" {
		t.Fatalf("missing DOT DIRS section; got:\n%s", combined)
	}
	for _, p := range paths {
		if strings.Contains(section, p) {
			t.Fatalf("DOT DIRS unexpectedly contains %q; section:\n%s", p, section)
		}
	}
}

func dotDirsSummarySection(combined string) string {
	idxPlan := strings.Index(combined, "dry-run: machine backup plan")
	if idxPlan < 0 {
		return ""
	}
	rest := combined[idxPlan:]
	idxDirs := strings.Index(rest, "  DOT DIRS")
	idxExcluded := strings.Index(rest, "  EXCLUDED")
	if idxDirs < 0 || idxExcluded < 0 || idxDirs >= idxExcluded {
		return ""
	}
	return rest[idxDirs:idxExcluded]
}

var dotDirSummaryRowRE = regexp.MustCompile(`^\s+(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB|GB))(?:\s+LARGE SIZE)?\s*$`)

func parseDotDirSummaryRows(section string) []struct {
	Path  string
	Files int
	Size  string
	Bytes int64
} {
	var rows []struct {
		Path  string
		Files int
		Size  string
		Bytes int64
	}
	for _, line := range strings.Split(section, "\n") {
		m := dotDirSummaryRowRE.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}
		files, _ := strconv.Atoi(m[2])
		rows = append(rows, struct {
			Path  string
			Files int
			Size  string
			Bytes int64
		}{Path: m[1], Files: files, Size: m[3], Bytes: parseHumanSizeToken(m[3])})
	}
	return rows
}

func parseHumanSizeToken(size string) int64 {
	size = strings.TrimSpace(size)
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB)$`)
	m := re.FindStringSubmatch(size)
	if len(m) < 3 {
		return 0
	}
	val, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	mult := int64(1)
	switch m[2] {
	case "KB":
		mult = 1024
	case "MB":
		mult = 1024 * 1024
	case "GB":
		mult = 1024 * 1024 * 1024
	}
	return int64(val * float64(mult))
}

func assertDotDirsSortedBySizeDesc(t *testing.T, combined string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	rows := parseDotDirSummaryRows(section)
	if len(rows) < 2 {
		t.Fatalf("want >=2 DOT DIRS summary rows for sort check; section:\n%s", section)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Bytes < rows[i].Bytes {
			t.Fatalf("DOT DIRS not sorted by size desc: %s (%d) before %s (%d); section:\n%s",
				rows[i-1].Path, rows[i-1].Bytes, rows[i].Path, rows[i].Bytes, section)
		}
		if rows[i-1].Bytes == rows[i].Bytes && rows[i-1].Path > rows[i].Path {
			t.Fatalf("DOT DIRS tiebreak not path asc at equal size: %s before %s; section:\n%s",
				rows[i-1].Path, rows[i].Path, section)
		}
	}
}

func assertSummaryDirHasLargeSize(t *testing.T, combined, dir string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, dir) {
			continue
		}
		if strings.Contains(line, "LARGE SIZE") {
			return
		}
		t.Fatalf("DOT DIRS row for %q missing LARGE SIZE; line:\n%s", dir, line)
	}
	t.Fatalf("DOT DIRS summary missing dir %q; section:\n%s", dir, section)
}

func assertSummaryDirLacksLargeSize(t *testing.T, combined, dir string) {
	t.Helper()
	section := dotDirsSummarySection(combined)
	if section == "" {
		t.Fatalf("missing summary DOT DIRS section; got:\n%s", combined)
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.Contains(line, dir) && strings.Contains(line, "LARGE SIZE") {
			t.Fatalf("DOT DIRS row for %q unexpectedly has LARGE SIZE; line:\n%s", dir, line)
		}
	}
}

func largeDirDetailSection(combined string) string {
	idx := strings.Index(combined, "LARGE DIR DETAIL:")
	if idx < 0 {
		return ""
	}
	rest := combined[idx+len("LARGE DIR DETAIL:"):]
	idxExcluded := strings.Index(rest, "  EXCLUDED")
	if idxExcluded < 0 {
		return rest
	}
	return rest[:idxExcluded]
}

var largeDirDetailLineRE = regexp.MustCompile(`^\s+>\s+(\S+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB|GB))\s*$`)

type largeDirDetailRow struct {
	Path  string
	Size  string
	Bytes int64
}

func parseLargeDirDetailLines(section string) []largeDirDetailRow {
	var rows []largeDirDetailRow
	for _, line := range strings.Split(section, "\n") {
		m := largeDirDetailLineRE.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		rows = append(rows, largeDirDetailRow{
			Path:  m[1],
			Size:  strings.TrimSpace(m[2]),
			Bytes: parseHumanSizeToken(m[2]),
		})
	}
	return rows
}

func assertLargeDirDetailFlatSorted(t *testing.T, combined string) []largeDirDetailRow {
	t.Helper()
	section := largeDirDetailSection(combined)
	if section == "" {
		t.Fatalf("missing LARGE DIR DETAIL section; got:\n%s", combined)
	}
	rows := parseLargeDirDetailLines(section)
	if len(rows) == 0 {
		t.Fatalf("LARGE DIR DETAIL has no flat detail lines; section:\n%s", section)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1].Bytes < rows[i].Bytes {
			t.Fatalf("LARGE DIR DETAIL not sorted by size desc: %s (%d) before %s (%d); section:\n%s",
				rows[i-1].Path, rows[i-1].Bytes, rows[i].Path, rows[i].Bytes, section)
		}
		if rows[i-1].Bytes == rows[i].Bytes && rows[i-1].Path > rows[i].Path {
			t.Fatalf("LARGE DIR DETAIL tiebreak not path asc at equal size: %s before %s; section:\n%s",
				rows[i-1].Path, rows[i].Path, section)
		}
	}
	return rows
}

func assertLargeDirDetailHasPaths(t *testing.T, combined string, paths ...string) {
	t.Helper()
	rows := assertLargeDirDetailFlatSorted(t, combined)
	seen := make(map[string]largeDirDetailRow, len(rows))
	for _, row := range rows {
		seen[row.Path] = row
	}
	for _, path := range paths {
		row, ok := seen[path]
		if !ok {
			t.Fatalf("LARGE DIR DETAIL missing path %q; rows=%v\nsection:\n%s", path, rows, largeDirDetailSection(combined))
		}
		if row.Bytes < largeDirDetailThresholdBytes {
			t.Fatalf("LARGE DIR DETAIL path %q size %s (%d B) below 10 MB detail threshold",
				path, row.Size, row.Bytes)
		}
	}
}

func assertLargeDirDetailLacksPaths(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := largeDirDetailSection(combined)
	if section == "" {
		return
	}
	rows := parseLargeDirDetailLines(section)
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		seen[row.Path] = struct{}{}
	}
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			t.Fatalf("LARGE DIR DETAIL unexpectedly lists %q; section:\n%s", path, section)
		}
	}
}

func assertLargeDirDetailContains(t *testing.T, combined string, dir string, children ...string) {
	t.Helper()
	paths := make([]string, 0, 1+len(children))
	paths = append(paths, dir)
	for _, child := range children {
		child = strings.TrimPrefix(child, "> ")
		child = strings.TrimSpace(child)
		if strings.Contains(child, "/") {
			paths = append(paths, child)
			continue
		}
		dirBase := strings.TrimSuffix(dir, "/")
		paths = append(paths, dirBase+"/"+child)
	}
	assertLargeDirDetailHasPaths(t, combined, paths...)
}

func assertNoLargeDirDetail(t *testing.T, combined string) {
	t.Helper()
	if strings.Contains(combined, "LARGE DIR DETAIL:") {
		t.Fatalf("unexpected LARGE DIR DETAIL section; got:\n%s", combined)
	}
}

func archiveUserMembers(members []string) []string {
	out := make([]string, 0, len(members))
	for _, m := range members {
		m = strings.TrimPrefix(m, "./")
		if m == "manifest.json" || strings.HasPrefix(m, ".backup/") {
			continue
		}
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func assertStringSetsEqual(t *testing.T, label string, want, got []string) {
	t.Helper()
	w := append([]string(nil), want...)
	g := append([]string(nil), got...)
	sort.Strings(w)
	sort.Strings(g)
	if len(w) != len(g) {
		t.Fatalf("%s count mismatch: want %d got %d\nwant=%v\ngot=%v", label, len(w), len(g), w, g)
	}
	for i := range w {
		if w[i] != g[i] {
			t.Fatalf("%s mismatch at %d: want %q got %q\nwant=%v\ngot=%v", label, i, w[i], g[i], w, g)
		}
	}
}

func assertExcludedMentions(t *testing.T, combined string, needles ...string) {
	t.Helper()
	section := excludedSection(combined)
	if section == "" {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	for _, n := range needles {
		if !strings.Contains(section, n) {
			t.Fatalf("EXCLUDED section missing %q; section:\n%s", n, section)
		}
	}
}

func assertDotFilesIncludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotFilesSection(combined)
	if section == "" {
		t.Fatalf("missing DOT FILES section; got:\n%s", combined)
	}
	for _, p := range paths {
		if !strings.Contains(section, p) {
			t.Fatalf("DOT FILES missing %q; section:\n%s", p, section)
		}
	}
}

func assertDotFilesExcludes(t *testing.T, combined string, paths ...string) {
	t.Helper()
	section := dotFilesSection(combined)
	if section == "" {
		t.Fatalf("missing DOT FILES section; got:\n%s", combined)
	}
	for _, p := range paths {
		if strings.Contains(section, p) {
			t.Fatalf("DOT FILES unexpectedly contains %q; section:\n%s", p, section)
		}
	}
}

func seedBackupMeta(t *testing.T, home string) {
	t.Helper()
	writeServerFile(t, home, ".backup/config.json", seededBackupMetaJSON)
}

func parseExclusionConfigJSON(t *testing.T, raw []byte) exclusionConfigJSON {
	t.Helper()
	var cfg exclusionConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse exclusion config: %v\n%s", err, raw)
	}
	return cfg
}

func writeServerFile(t *testing.T, home, rel, content string) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func readServerFile(t *testing.T, home, rel string) string {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", full, err)
	}
	return string(data)
}

func tarXZListMembers(t *testing.T, archivePath string) []string {
	t.Helper()
	cmd := exec.Command("tar", "-tJf", archivePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar -tJf %s: %v\n%s", archivePath, err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func tarXZExtractFile(t *testing.T, archivePath, member string) []byte {
	t.Helper()
	dir, err := os.MkdirTemp("", "machine-backup-extract-*")
	if err != nil {
		t.Fatalf("mkdir extract: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	cmd := exec.Command("tar", "-xJf", archivePath, "-C", dir, member)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar extract %s from %s: %v\n%s", member, archivePath, err, out)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(member)))
	if err != nil {
		t.Fatalf("read extracted %s: %v", member, err)
	}
	return data
}

func archiveHasXZMagic(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if len(data) < 6 {
		t.Fatalf("archive too short (%d bytes)", len(data))
	}
	magic := []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}
	if !bytes.Equal(data[:6], magic) {
		t.Fatalf("missing xz magic in %s (got % x)", path, data[:6])
	}
}

func memberListContains(members []string, want string) bool {
	want = strings.TrimPrefix(want, "./")
	for _, m := range members {
		m = strings.TrimPrefix(m, "./")
		if m == want {
			return true
		}
	}
	return false
}

func combinedHasAll(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(combined, n) {
			t.Fatalf("output missing %q;\nhave:\n%s", n, combined)
		}
	}
}

func assertCacheExclusionReason(t *testing.T, combined string) {
	t.Helper()
	idx := strings.Index(combined, "EXCLUDED")
	if idx < 0 {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	section := combined[idx:]
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		if !strings.Contains(line, ".cache") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "cache") || strings.Contains(lower, "temporary") {
			return
		}
	}
	t.Fatalf("EXCLUDED section missing reason for .cache; got:\n%s", section)
}

func combinedHasNone(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(combined, n) {
			t.Fatalf("output unexpectedly contains %q;\nhave:\n%s", n, combined)
		}
	}
}

func paddedBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'x'
	}
	return b
}

// seedExcludedSizesFixtures overwrites cache/log fixtures with known byte sizes so
// EXCLUDED per-rule FILES and SIZE columns are deterministic (1024+512 B under .cache,
// 512 B under **/*.log).
func seedExcludedSizesFixtures(t *testing.T, home string) {
	t.Helper()
	writeServerBinaryFile(t, home, ".cache/junk", paddedBytes(1024))
	writeServerBinaryFile(t, home, ".cache/nested/deep", paddedBytes(512))
	writeServerBinaryFile(t, home, ".ai-critic/service.log", paddedBytes(512))
}

var (
	backupSizeToken       = regexp.MustCompile(`\d+(\.\d+)?\s*(B|KB|MB)\b`)
	excludedStatsHeaderRE = regexp.MustCompile(`EXCLUDED \(\d+ paths, \d+ files,`)
	excludedRuleRowRE     = regexp.MustCompile(`^\s+(\S.*?)\s+(\d+)\s+(\d+(?:\.\d+)?\s*(?:B|KB|MB))\s+`)
)

func assertExcludedStatsHeader(t *testing.T, combined string) {
	t.Helper()
	section := excludedSection(combined)
	if section == "" {
		t.Fatalf("missing EXCLUDED section; got:\n%s", combined)
	}
	if !excludedStatsHeaderRE.MatchString(section) {
		t.Fatalf("EXCLUDED section missing paths/files/size header; section:\n%s", section)
	}
	if !backupSizeToken.MatchString(section) {
		t.Fatalf("EXCLUDED header missing size token; section:\n%s", section)
	}
}

func assertExcludedTableHeaders(t *testing.T, combined string) {
	t.Helper()
	section := excludedSection(combined)
	if !strings.Contains(section, "RULE") || !strings.Contains(section, "FILES") {
		t.Fatalf("EXCLUDED section missing RULE/FILES column headers; section:\n%s", section)
	}
}

func excludedRuleLineIndex(section, rule string) int {
	lines := strings.Split(section, "\n")
	for i, line := range lines {
		if strings.Contains(line, rule) && excludedRuleRowRE.MatchString(line) {
			return i
		}
	}
	return -1
}

func assertExcludedRuleBefore(t *testing.T, combined, firstRule, secondRule string) {
	t.Helper()
	section := excludedSection(combined)
	i := excludedRuleLineIndex(section, firstRule)
	j := excludedRuleLineIndex(section, secondRule)
	if i < 0 {
		t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", firstRule, section)
	}
	if j < 0 {
		t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", secondRule, section)
	}
	if i >= j {
		t.Fatalf("expected %q row before %q row (lines %d vs %d); section:\n%s", firstRule, secondRule, i, j, section)
	}
}

func parseExcludedRuleRow(line string) (rule string, files int, size string, ok bool) {
	m := excludedRuleRowRE.FindStringSubmatch(line)
	if len(m) < 4 {
		return "", 0, "", false
	}
	files, err := strconv.Atoi(m[2])
	if err != nil {
		return "", 0, "", false
	}
	return strings.TrimSpace(m[1]), files, strings.TrimSpace(m[3]), true
}

func assertExcludedRuleFilesAtLeast(t *testing.T, combined, rule string, minFiles int) {
	t.Helper()
	section := excludedSection(combined)
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, rule) {
			continue
		}
		gotRule, files, _, ok := parseExcludedRuleRow(line)
		if !ok || !strings.Contains(gotRule, rule) {
			continue
		}
		if files >= minFiles {
			return
		}
		t.Fatalf("EXCLUDED rule %q files=%d, want >= %d; line:\n%s", rule, files, minFiles, line)
	}
	t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", rule, section)
}

func assertExcludedRuleHasSizeKB(t *testing.T, combined, rule string) {
	t.Helper()
	section := excludedSection(combined)
	for _, line := range strings.Split(section, "\n") {
		if !strings.Contains(line, rule) {
			continue
		}
		gotRule, _, size, ok := parseExcludedRuleRow(line)
		if !ok || !strings.Contains(gotRule, rule) {
			continue
		}
		if strings.Contains(size, "KB") || strings.Contains(size, "MB") {
			return
		}
		t.Fatalf("EXCLUDED rule %q size=%q, want >= 1 KB; line:\n%s", rule, size, line)
	}
	t.Fatalf("EXCLUDED section missing rule row %q; section:\n%s", rule, section)
}

type backupPlanFetchBody struct {
	DryRun                 bool     `json:"dry_run"`
	Exclude                []string `json:"exclude"`
	Include                []string `json:"include"`
	LargeDirThresholdBytes int64    `json:"large_dir_threshold_bytes,omitempty"`
}

func fetchBackupPlanIncluded(t *testing.T, serverURL, token string, exclude, include []string, largeDirThreshold int64) []string {
	t.Helper()
	body := backupPlanFetchBody{DryRun: true, Exclude: exclude, Include: include}
	if largeDirThreshold > 0 {
		body.LargeDirThresholdBytes = largeDirThreshold
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal backup plan body: %v", err)
	}
	backupURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/remote-agent/machine/backup"
	req, err := http.NewRequest(http.MethodPost, backupURL, bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("build backup plan request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch backup plan: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read backup plan response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backup plan status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var plan struct {
		Included []string `json:"included"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("decode backup plan: %v\n%s", err, data)
	}
	return plan.Included
}

func subcommandFlagsFromRequest(req *Request) []string {
	var flags []string
	for _, ex := range req.ExcludePaths {
		flags = append(flags, "--exclude", ex)
	}
	for _, inc := range req.IncludePaths {
		flags = append(flags, "--include", inc)
	}
	if req.ShowConfig {
		flags = append(flags, "--show-config")
	}
	if req.ShowMeta {
		flags = append(flags, "--show-meta")
	}
	if req.SetConfig {
		flags = append(flags, "--set-config")
	}
	if req.SetConfigLargeDirThreshold != "" {
		flags = append(flags, "--large-dir-threshold", req.SetConfigLargeDirThreshold)
	}
	if req.SkipGitDirsScan {
		flags = append(flags, "--skip-git-dirs-scan")
	}
	if req.GitDirsScanMaxDepth > 0 {
		flags = append(flags, "--git-dirs-scan-max-depth", strconv.Itoa(req.GitDirsScanMaxDepth))
	}
	return flags
}

func argvWithoutDryRun(argv []string) []string {
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		if argv[i] == "--dry-run" {
			continue
		}
		out = append(out, argv[i])
	}
	return out
}

func runDryRunThenArchive(t *testing.T, req *Request, resp *Response, agentBin string, agentEnv []string, serverURL string) (*Response, error) {
	dryArgv := make([]string, 0, len(req.Args)+16)
	dryArgv = append(dryArgv, "--server", serverURL, "--token", req.Token)
	dryArgv = append(dryArgv, req.Args...)
	dryArgv = insertSubcommandFlags(dryArgv, subcommandFlagsFromRequest(req)...)

	t.Logf("dry-run argv: %v", dryArgv)
	exitCode, stdout, stderr, runErr := runAgent(agentBin, dryArgv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.DryRunCombined = strings.TrimSpace(stdout + "\n" + stderr)
	resp.Combined = resp.DryRunCombined
	if exitCode != 0 {
		return resp, nil
	}

	resp.DryRunIncluded = fetchBackupPlanIncluded(t, serverURL, req.Token, req.ExcludePaths, req.IncludePaths, 0)

	if req.OutputPath == "" {
		return nil, fmt.Errorf("DryRunThenArchive requires OutputPath")
	}
	out := req.OutputPath
	if !filepath.IsAbs(out) {
		out = filepath.Join(resp.AgentHome, out)
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return nil, err
	}
	req.OutputPath = absOut

	backupArgv := argvWithoutDryRun(dryArgv)
	backupArgv = insertSubcommandFlags(backupArgv, "--output", absOut)
	t.Logf("backup argv: %v", backupArgv)
	exitCode, stdout, stderr, runErr = runAgent(agentBin, backupArgv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}
	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(stdout + "\n" + stderr)
	resp.BackupPath = absOut
	return resp, nil
}
```