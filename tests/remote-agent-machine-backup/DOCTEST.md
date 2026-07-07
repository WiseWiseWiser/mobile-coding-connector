# Remote-Agent Machine Backup & Restore Doctests

End-to-end tests for `remote-agent machine backup` and `remote-agent machine restore`:
server-home dot-file/dot-dir discovery, built-in and custom exclusions, SSE
streaming dry-run plans (per-entry sizes + summary), streamed `tar.xz` archives
with `manifest.json`, and restore with identical skip reporting.

# DSN (Domain Specific Notion)

The harness models a remote machine as an isolated `serverHome` directory. The
`ai-critic-server` subprocess runs with `HOME=serverHome` and working directory
`serverHome`, so server `~` and backup scope align. The CLI runs in a separate
`agentHome` with only `remote-agent-config.json`. Backup walks direct children of
server home (dot-files and dot-dirs), applies built-in exclusions plus optional
`--exclude`, archives symlinks without following, and streams `tar.xz` for real
backups. With `--dry-run`, backup and restore use SSE `/stream` endpoints:
incremental per-entry lines (with sizes on backup) followed by a human summary
block after the `done` frame. Restore reads the archive, skips byte-identical
entries (printing `skip (identical): <path>` in dry-run and apply), and applies
create/update actions.

**Participants**

- **remote-agent subprocess** — `./cmd/remote-agent`; subcommands `machine backup`
  and `machine restore` with `--server` / `--token`.
- **ai-critic-server subprocess** — ephemeral port; `POST /api/remote-agent/machine/backup`
  (`application/x-xz` stream) and `POST /api/remote-agent/machine/backup/stream` (SSE
  dry-run plan); `POST /api/remote-agent/machine/restore` (apply) and
  `POST /api/remote-agent/machine/restore/stream?dry_run=true` (SSE dry-run plan).
- **serverHome** — temp fake machine home seeded with dot fixtures, built-in
  exclusion trees (`.cache`, `.npm`, `.cargo/registry`, etc.), and v1.1 extended
  fixtures (ELF stub, `.log` files, `upload-chunks/`, SQLite DB, JPEG image, new
  path-prefix trees).
- **agentHome** — temp `HOME` for `~/.ai-critic/remote-agent-config.json` only.
- **session cache** — doctest-injected `DOCTEST_SESSION_ID` keys
  `$TMPDIR/machine-backup-doctest-<id>/` for shared binaries and a default prereq
  archive (file lock + flock). Helpers use the variable directly, not `os.Getenv`.

**Behaviors**

- `machine backup --dry-run` streams DOT FILES / DOT DIRS / EXCLUDED sections with
  per-entry sizes (included paths) and per-rule FILES/SIZE/REASON table (excluded
  rules, one line per rule sorted by SIZE descending), then prints
  `dry-run: machine backup plan` summary with DOT FILES/DOT DIRS sorted by size
  descending (path tiebreak), `LARGE SIZE` flags for included dirs over threshold
  (default 40 MB, overridable via `--large-dir-threshold`), and a flat `LARGE DIR DETAIL`
  list of every included directory ≥ 10 MB (size descending, path tiebreak; parent and
  child rows may both appear; excluded trees omitted); no archive file is written. Real `machine backup` archives
  the same included path set the plan reports (`.backup/` meta injected at pack time).
- `machine backup --show-config` prints effective merged exclusion config JSON from
  the server (builtin + `~/.ai-critic/backup-config.json` + optional CLI flags),
  with display reason `from user config` for user paths whose persisted `reason`
  is empty/omitted and `user excluded` for paths added only via CLI `--exclude`.
  Bare `--show-config` omits CLI; repeatable `--exclude` / `--include` /
  `--large-dir-threshold` forward to GET backup-config query params for preview.
- `machine backup --set-config --exclude PATH...` and/or `--large-dir-threshold SIZE`
  persists user-authored fields to `~/.ai-critic/backup-config.json` on the server
  (CLI-set excludes omit `reason`; stdout prints effective merged JSON + trailing newline).
  Bare `--set-config` or combinations with `--dry-run` / `--show-config` / `--output` /
  `--include` exit non-zero.
- Persisted `large_dir_threshold` resolves at runtime when CLI omits `--large-dir-threshold`
  (CLI per-run value wins, then user config, then 40 MB default).
- `machine backup` (default) streams `tar.xz` containing `manifest.json`, included
  paths relative to server home, and phantom `.backup/` meta entries injected at
  pack time (`config.json`, `installed.json`, `ENV`, optional `git-repo-worktrees.json`
  when git scan runs, and optional `*.machine.bak` snapshots of pre-existing
  `~/.backup/*` files).
- Repeatable `--exclude` merges with built-in exclusions; repeatable `--include`
  re-includes built-in excluded paths (exact path overrides for `.log` suffix and
  `**(binary)` rules). Effective rule: `(defaults − include) ∪ exclude`.
- Built-in exclusion config version `1.1` adds path-prefix entries, segment rule
  `**/upload-chunks`, suffix rule `**/*.log`, and executable rule `**(binary)`
  (`IsExecutableBinary` for ELF/Mach-O/PE only; SQLite and images stay included).
- `machine restore --dry-run` streams `skip (identical):` / `update:` / `create:` lines,
  then prints `dry-run: machine restore plan` summary with counts; no writes.
- `machine restore` applies create/update entries; identical paths are skipped with
  the same skip line printed to stdout.
- `machine restore --show-config` without archive prints effective merged config
  JSON from the server (same CLI merge as backup preview); with archive prints
  `.backup/config.json` from the archive (or effective merged fallback; no CLI merge).
- `machine backup` discovers git repos under included top-level dot-dirs via
  `scan_repo`, enriches branch/short-sha/status/worktrees, and prints a `GIT REPOS`
  section in the dry-run summary (`(none)` when no repos, `(skipped)` with
  `--skip-git-dirs-scan`). Per-repo and per-root scan/enrichment failures are
  recorded as `error:` lines in GIT REPOS; backup and dry-run never abort on git
  meta scan failures. `--git-dirs-scan-max-depth N` caps scan depth (`0` =
  unlimited). Real backups embed `.backup/git-repo-worktrees.json` when scan runs.
- `machine restore --show-meta` requires an archive and prints `.backup/*` meta
  except `config.json` and `*.machine.bak` (includes `git-repo-worktrees.json`
  when present in the archive).
- Restore skips meta snapshots (`.backup/config.json`, `.backup/installed.json`,
  `.backup/ENV`) but restores `.backup/*.machine.bak` to `~/.backup/{original name}`.

## Version

0.0.2

## Decision Tree

```
[remote-agent machine backup | restore]
 |
 +-- backup/                              (GROUP)  snapshot server HOME
 |    |
 |    +-- dry-run/                        (LEAF)   plan only; rollups + exclusion reasons
 |    +-- excluded-sizes/                 (LEAF)   EXCLUDED per-rule FILES/SIZE table + sort
 |    +-- stream/                         (LEAF)   tar.xz stream; manifest + members
 |    +-- custom-exclude/                 (LEAF)   --exclude drops extra dot-dir
 |    +-- show-config/                    (LEAF)   effective merged exclusion config JSON (v1.1)
 |    +-- set-config/                     (LEAF)   persist user excludes (empty reason) to backup-config.json
 |    +-- set-config-threshold/           (LEAF)   threshold-only set-config preserves prior excludes
 |    +-- set-config-merge/              (LEAF)   incremental --exclude unions into persisted exclude_paths
 |    +-- set-config-merge-threshold/    (LEAF)   exclude-only set-config preserves persisted threshold
 |    +-- set-config-empty/               (LEAF)   bare --set-config errors (non-zero exit)
 |    +-- set-config-mutual-exclude/      (LEAF)   --set-config --dry-run errors (non-zero exit)
 |    +-- persisted-merge/               (LEAF)   persisted excludes merge into dry-run + show-config display
 |    +-- persisted-threshold/           (LEAF)   persisted threshold suppresses LARGE SIZE without CLI flag
 |    +-- show-config-persisted/         (LEAF)   show-config display reasons for user config paths
 |    +-- show-config-cli-exclude/       (LEAF)   show-config --exclude merges CLI with reason user excluded
 |    +-- show-config-cli-include/       (LEAF)   show-config --include drops path from effective exclude_paths
 |    +-- extended-exclusions/            (LEAF)   v1.1 special rules + new path entries
 |    +-- upload-chunks/                  (LEAF)   **/upload-chunks segment excluded
 |    +-- log-suffix/                     (LEAF)   **/*.log suffix excluded
 |    +-- include-log/                    (LEAF)   --include re-includes specific .log
 |    +-- binary-exclude/                 (LEAF)   **(binary) excludes ELF stub
 |    +-- include-binary/                 (LEAF)   --include re-includes executable
 |    +-- keep-sqlite/                    (LEAF)   SQLite opencode.db stays in archive
 |    +-- keep-images/                    (LEAF)   .live-and-love/imgs/*.jpg included
 |    +-- path-exclusions/                (LEAF)   five path-prefix trees omitted
 |    +-- include/                        (LEAF)   --include re-includes built-in path
 |    +-- backup-meta/                    (LEAF)   archive .backup/ meta + machine.bak
 |    +-- large-dir-summary/              (LEAF)   LARGE SIZE + flat LARGE DIR DETAIL + size sort
 |    +-- large-dir-detail-deep/          (LEAF)   deep scan lists nested ≥10 MB dirs; excludes .cache
 |    +-- large-dir-threshold/            (LEAF)   --large-dir-threshold suppresses LARGE SIZE only
 |    +-- dry-run-matches-archive/        (LEAF)   dry-run included set == tar members
 |    +-- included-fetch-skills/          (LEAF)   reverted exclusions stay included
 |    +-- git-repos-summary/              (LEAF)   GIT REPOS dry-run lists main repo
 |    +-- git-repos-worktree/             (LEAF)   nested worktree + dirty status
 |    +-- git-repos-empty-repo/           (LEAF)   init-only repo → error line; exit 0
 |    +-- git-repos-none/                 (LEAF)   GIT REPOS: (none)
 |    +-- git-repos-skipped/              (LEAF)   skip scan; no archive JSON
 |    +-- git-repos-max-depth/            (LEAF)   max depth excludes deep repo
 |    +-- git-repos-archive/              (LEAF)   archive git-repo-worktrees.json
 |
 +-- restore/                             (GROUP)  apply archive to server HOME
      |
      +-- dry-run-identical/              (LEAF)   unchanged home → skip lines only
      +-- dry-run-changed/                (LEAF)   mutated file → update in plan
      +-- apply/                          (LEAF)   writes changes; skips identical
      +-- show-config-builtin/            (LEAF)   effective merged config JSON (no archive)
      +-- show-config-cli-exclude/        (LEAF)   show-config --exclude merges CLI (no archive)
      +-- show-config-archive/            (LEAF)   effective config from archive
      +-- show-meta/                      (LEAF)   installed.json + ENV sections
      +-- show-meta-git-repos/            (LEAF)   --show-meta prints git JSON
      +-- meta-restore/                   (LEAF)   .machine.bak restores ~/.backup/*
```

## Test Index

| # | Leaf | Description |
|---|------|-------------|
| 1 | `backup/dry-run` | Streamed plan with sizes and exclusion reasons; summary rollups |
| 2 | `backup/excluded-sizes` | EXCLUDED header totals + per-rule FILES/SIZE table sorted by size |
| 3 | `backup/stream` | Writes valid `tar.xz` with manifest and included members |
| 4 | `backup/custom-exclude` | `--exclude .docker` omits `.docker` from plan and archive |
| 5 | `backup/show-config` | `--show-config` prints effective merged exclusion JSON v1.1 |
| 28 | `backup/set-config` | `--set-config` writes excludes with empty/omitted reason (not `user excluded`) |
| 29 | `backup/persisted-merge` | Persisted excludes merge into dry-run; show-config shows `from user config` |
| 30 | `backup/set-config-threshold` | Threshold-only set-config persists `100MB` without wiping prereq excludes |
| 38 | `backup/set-config-merge` | Prereq `.knowledge-hub`; second set-config adds `.docker`; file has both |
| 39 | `backup/set-config-merge-threshold` | Prereq exclude + `50MB` threshold; exclude-only set-config preserves threshold |
| 31 | `backup/set-config-empty` | Bare `--set-config` exits non-zero with error message |
| 32 | `backup/set-config-mutual-exclude` | `--set-config --dry-run` exits non-zero |
| 33 | `backup/persisted-threshold` | Prereq 100 MB threshold; dry-run without CLI flag suppresses LARGE SIZE |
| 34 | `backup/show-config-persisted` | Show-config: `from user config` vs preserved manual reason |
| 35 | `backup/show-config-cli-exclude` | `--show-config --exclude .knowledge-index` → reason `user excluded` |
| 36 | `backup/show-config-cli-include` | Prereq exclude `.cache`; `--show-config --include .cache` omits `.cache` |
| 6 | `backup/extended-exclusions` | v1.1 lists `**(binary)`, `**/*.log`, `**/upload-chunks`, six new paths |
| 7 | `backup/upload-chunks` | `upload-chunks` segment trees excluded in dry-run EXCLUDED |
| 8 | `backup/log-suffix` | `*.log` files excluded; non-log config files remain included |
| 9 | `backup/include-log` | `--include` re-includes a specific `.log` file |
| 10 | `backup/binary-exclude` | ELF executable excluded via `**(binary)` rule |
| 11 | `backup/include-binary` | `--include` re-includes a specific executable binary |
| 12 | `backup/keep-sqlite` | SQLite `opencode.db` included in archive (not binary rule) |
| 13 | `backup/keep-images` | `.live-and-love/imgs/*.jpg` included in archive |
| 14 | `backup/path-exclusions` | Five path-prefix trees omitted; confluence-fetch data included |
| 15 | `backup/include` | `--include .cache` re-includes `.cache` tree in dry-run plan |
| 16 | `backup/backup-meta` | Archive contains `.backup/` meta; seeded config → `.machine.bak` |
| 17 | `backup/large-dir-summary` | Default threshold flags `.big-test`; flat LARGE DIR DETAIL lists parent + children |
| 40 | `backup/large-dir-detail-deep` | Nested `.deep-test/nested-big` (12 MB) in flat detail; `.cache` absent |
| 18 | `backup/large-dir-threshold` | `--large-dir-threshold 100MB` hides LARGE SIZE; detail still lists ≥10 MB dirs |
| 19 | `backup/dry-run-matches-archive` | Plan included paths match archive members (minus `.backup/` meta) |
| 20 | `backup/included-fetch-skills` | git-fetch, confluence-fetch, knowledge-index paths in dry-run + archive |
| 21 | `restore/dry-run-identical` | Home matches archive → skip stream + restore summary, no writes |
| 22 | `restore/dry-run-changed` | Modified `.bashrc` → `update:` stream line + restore summary counts |
| 23 | `restore/apply` | Apply restores changed file; identical paths still skipped |
| 24 | `restore/show-config-builtin` | `--show-config` without archive prints effective merged JSON v1.1 |
| 37 | `restore/show-config-cli-exclude` | Restore `--show-config --exclude .knowledge-index` merges CLI (no archive) |
| 25 | `restore/show-config-archive` | Prereq backup → `--show-config` prints archive effective config |
| 26 | `restore/show-meta` | Prereq backup → `--show-meta` prints installed.json + ENV (not git JSON) |
| 27 | `restore/meta-restore` | Prereq backup with seeded meta → apply restores `.machine.bak` content |
| 41 | `backup/git-repos-summary` | Dry-run GIT REPOS lists `.wrk-test/main` (branch, 7-char sha, clean, commit msg) |
| 42 | `backup/git-repos-worktree` | Dry-run GIT REPOS nests worktree with dirty status count |
| 48 | `backup/git-repos-empty-repo` | Init-only `.wrk-test/empty` → `error: no commits (HEAD unborn)`; exit 0 |
| 43 | `backup/git-repos-none` | Default seed → `GIT REPOS: (none)` |
| 44 | `backup/git-repos-skipped` | `--skip-git-dirs-scan` → `(skipped)`; archive omits git JSON |
| 45 | `backup/git-repos-max-depth` | Deep repo beyond `--git-dirs-scan-max-depth 2` → `(none)` |
| 46 | `backup/git-repos-archive` | Real backup archive contains valid `git-repo-worktrees.json` |
| 47 | `restore/show-meta-git-repos` | Prereq backup with git → `--show-meta` prints git JSON section |

## Parameter Coverage

| Factor | Leaves |
|--------|--------|
| Subcommand `backup` | backup/* |
| Subcommand `restore` | restore/* |
| `--dry-run` | backup/dry-run, backup/excluded-sizes, backup/large-dir-summary, backup/large-dir-detail-deep, backup/large-dir-threshold, backup/dry-run-matches-archive, backup/included-fetch-skills, backup/upload-chunks, backup/log-suffix, backup/include-log, backup/binary-exclude, backup/include-binary, backup/include, restore/dry-run-identical, restore/dry-run-changed |
| `--large-dir-threshold` | backup/large-dir-threshold, backup/set-config-threshold, backup/persisted-threshold |
| `--set-config` | backup/set-config, backup/set-config-threshold, backup/set-config-merge, backup/set-config-merge-threshold, backup/set-config-empty, backup/set-config-mutual-exclude, backup/persisted-merge, backup/persisted-threshold, backup/show-config-persisted |
| Set-config incremental merge | backup/set-config-merge, backup/set-config-merge-threshold, backup/set-config-threshold |
| Persisted backup-config merge (runtime) | backup/persisted-merge, backup/persisted-threshold, backup/show-config-persisted |
| Set-config validation errors | backup/set-config-empty, backup/set-config-mutual-exclude |
| Dry-run ≡ archive invariant | backup/dry-run-matches-archive |
| Reverted exclusions (fetch skills, knowledge-index) | backup/included-fetch-skills, backup/path-exclusions, backup/extended-exclusions |
| LARGE SIZE / LARGE DIR DETAIL summary | backup/large-dir-summary, backup/large-dir-detail-deep, backup/large-dir-threshold, backup/persisted-threshold |
| `--show-config` | backup/show-config, backup/show-config-persisted, backup/persisted-merge, backup/extended-exclusions, backup/show-config-cli-exclude, backup/show-config-cli-include, restore/show-config-builtin, restore/show-config-cli-exclude, restore/show-config-archive |
| `--show-config` + CLI merge (`--exclude` / `--include`) | backup/show-config-cli-exclude, backup/show-config-cli-include, restore/show-config-cli-exclude |
| `--show-meta` | restore/show-meta |
| Streamed archive (no dry-run) | backup/stream, backup/keep-sqlite, backup/keep-images, backup/path-exclusions, backup/backup-meta, restore/* (prereq backup except show-config-builtin) |
| Built-in exclusions v1.1 | backup/dry-run, backup/excluded-sizes, backup/stream, backup/show-config, backup/extended-exclusions, backup/upload-chunks, backup/log-suffix, backup/binary-exclude, backup/path-exclusions, restore/show-config-builtin |
| EXCLUDED per-rule FILES/SIZE stats | backup/excluded-sizes, backup/dry-run (header totals) |
| Special rules `**/upload-chunks`, `**/*.log`, `**(binary)` | backup/upload-chunks, backup/log-suffix, backup/binary-exclude |
| Per-file `--include` overrides | backup/include-log, backup/include-binary, backup/include, backup/show-config-cli-include, restore/meta-restore (`--include .backup` for machine.bak apply) |
| Keep SQLite / images | backup/keep-sqlite, backup/keep-images |
| Custom `--exclude` | backup/custom-exclude, backup/show-config-cli-exclude, restore/show-config-cli-exclude |
| Archive `.backup/` meta | backup/backup-meta, restore/show-config-archive, restore/show-meta, restore/meta-restore |
| Seeded `~/.backup/config.json` | backup/backup-meta, restore/meta-restore |
| Identical vs changed restore target | restore/dry-run-identical, restore/dry-run-changed, restore/apply, restore/meta-restore |
| `--skip-git-dirs-scan` | backup/git-repos-skipped |
| `--git-dirs-scan-max-depth` | backup/git-repos-max-depth |
| Git repo scan / `GIT REPOS` summary | backup/git-repos-summary, backup/git-repos-worktree, backup/git-repos-empty-repo, backup/git-repos-none, backup/git-repos-skipped, backup/git-repos-max-depth |
| Per-repo git enrichment error (durable) | backup/git-repos-empty-repo |
| `.backup/git-repo-worktrees.json` | backup/git-repos-archive, backup/git-repos-skipped (absent), restore/show-meta-git-repos |
| Git fixture seeding (`SeedGitRepos*`) | backup/git-repos-summary, backup/git-repos-worktree, backup/git-repos-empty-repo, backup/git-repos-skipped, backup/git-repos-archive, restore/show-meta-git-repos |

## How to Run

```sh
go run ./script/build
doctest vet ./tests/remote-agent-machine-backup
doctest test -v ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
go test ./dot-pkgs-with-critic/go-pkgs/file/detect/... -count=1
```

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

type PostSetConfigExcludeEntry struct {
	Path   string
	Reason string
}

type Request struct {
	Args  []string
	Server string
	Token  string

	// OutputPath is the backup archive destination (backup leaves).
	OutputPath string

	// RestoreArchive is the tar.xz path passed to restore (set by Run when PrereqBackup).
	RestoreArchive string

	// ExcludePaths are appended as repeated --exclude flags before Args.
	ExcludePaths []string

	// IncludePaths are appended as repeated --include flags before Args.
	IncludePaths []string

	// PrereqBackup causes Run to execute `machine backup` before the main invocation.
	PrereqBackup bool

	// AfterBackupMutate selects post-backup server home changes for restore leaves.
	// Values: "" (none), "modify-bashrc", "wipe-backup-config".
	AfterBackupMutate string

	// SeedDocker adds .docker/config for custom-exclude coverage.
	SeedDocker bool

	// SeedBackupMeta seeds serverHome/.backup/config.json with distinguishable old JSON.
	SeedBackupMeta bool

	// SeedExcludedSizes overwrites cache/log fixtures with known byte sizes for
	// EXCLUDED per-rule FILES/SIZE assertions.
	SeedExcludedSizes bool

	// SeedLargeDir adds .big-test/ (>40 MB) and .small-test/ for large-dir summary leaves.
	SeedLargeDir bool

	// SeedLargeDirDetailDeep adds SeedLargeDir plus .deep-test/nested-big/ (12 MB) and small sibling.
	SeedLargeDirDetailDeep bool

	// SeedIncludedFetchSkills adds files under git-fetch, confluence-fetch, knowledge-index.
	SeedIncludedFetchSkills bool

	// DryRunThenArchive runs CLI dry-run then CLI backup; populates DryRunIncluded.
	DryRunThenArchive bool

	// ShowConfig and ShowMeta are set by leaves; Run appends flags when building argv.
	ShowConfig bool
	ShowMeta   bool

	// SetConfig appends --set-config (persist ExcludePaths to backup-config.json).
	SetConfig bool

	// PrereqSetConfig runs machine backup --set-config before the main invocation.
	PrereqSetConfig bool

	// SetConfigExcludePaths supplies --exclude for --set-config / PrereqSetConfig when
	// the main invocation should not repeat them.
	SetConfigExcludePaths []string

	// SetConfigLargeDirThreshold is passed as --large-dir-threshold with --set-config.
	SetConfigLargeDirThreshold string

	// PrereqSetConfigLargeDirThreshold supplies threshold for PrereqSetConfig.
	PrereqSetConfigLargeDirThreshold string

	// PostPrereqSetConfigExcludes patches persisted backup-config.json after PrereqSetConfig.
	PostPrereqSetConfigExcludes []PostSetConfigExcludeEntry

	// FollowUpShowConfig runs --show-config after the main invocation (FollowUpStdout).
	FollowUpShowConfig bool

	// SeedKnowledgeHub adds .knowledge-hub and .knowledge-index fixtures.
	SeedKnowledgeHub bool

	// SeedGitRepos adds a git fixture under .wrk-test/main (included dot-dir).
	SeedGitRepos bool

	// SeedGitReposWorktree also adds a linked worktree + dirty file in worktree.
	SeedGitReposWorktree bool

	// SeedGitReposMaxDepth nests a repo at .wrk-test/a/b/c/deep-repo for max-depth leaves.
	SeedGitReposMaxDepth bool

	// SeedGitReposEmpty adds git init only under .wrk-test/empty (no commits).
	SeedGitReposEmpty bool

	// SkipGitDirsScan sets --skip-git-dirs-scan on backup invocation.
	SkipGitDirsScan bool

	// GitDirsScanMaxDepth sets --git-dirs-scan-max-depth (0 = omit flag).
	GitDirsScanMaxDepth int
}

type Response struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Combined   string
	ServerPort int
	ServerHome string
	AgentHome  string

	BackupPath string

	// DryRunCombined is stdout from the dry-run leg when DryRunThenArchive is set.
	DryRunCombined string

	// DryRunIncluded is the server plan included path set (JSON API, same flags).
	DryRunIncluded []string

	// FollowUpStdout is stdout from a follow-up --show-config when FollowUpShowConfig is set.
	FollowUpStdout string
}

func Run(t *testing.T, req *Request) (*Response, error) {
	resp := &Response{}

	if req.Token == "" {
		req.Token = lib.TestPassword
	}

	moduleRoot := findModuleRoot()
	cacheDir := sessionCacheDir()
	serverBin, agentBin := buildSessionBinariesOnce(t, moduleRoot, cacheDir)

	serverHome, err := os.MkdirTemp("", "machine-backup-server-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(serverHome) })
	resp.ServerHome = serverHome

	if err := seedServerHome(t, serverHome, req.SeedDocker); err != nil {
		return nil, err
	}
	if req.SeedBackupMeta {
		seedBackupMeta(t, serverHome)
	}
	if req.SeedExcludedSizes {
		seedExcludedSizesFixtures(t, serverHome)
	}
	if req.SeedLargeDir {
		seedLargeDirFixture(t, serverHome)
		seedSmallDirForSortFixture(t, serverHome)
	}
	if req.SeedLargeDirDetailDeep {
		seedLargeDirDetailDeepFixture(t, serverHome)
	}
	if req.SeedIncludedFetchSkills {
		seedIncludedFetchSkills(t, serverHome)
	}
	if req.SeedKnowledgeHub {
		seedKnowledgeHub(t, serverHome)
	}
	if req.SeedGitReposWorktree {
		seedGitReposWorktreeFixture(t, serverHome)
	} else if req.SeedGitRepos {
		seedGitReposFixture(t, serverHome)
	} else if req.SeedGitReposEmpty {
		seedGitReposEmptyFixture(t, serverHome)
	}
	if req.SeedGitReposMaxDepth {
		seedGitReposMaxDepthFixture(t, serverHome)
	}

	agentHome, err := os.MkdirTemp("", "machine-backup-agent-home-*")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { os.RemoveAll(agentHome) })
	resp.AgentHome = agentHome

	credDir := filepath.Join(serverHome, ".ai-critic")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return nil, err
	}
	credFile := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credFile, []byte(req.Token+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("write credentials: %w", err)
	}

	remoteConfigPath := filepath.Join(agentHome, ".ai-critic", "remote-agent-config.json")
	if err := os.MkdirAll(filepath.Dir(remoteConfigPath), 0755); err != nil {
		return nil, err
	}

	portBase := portBaseFromTestName(t.Name())
	serverPort := pickFreePort(portBase)
	resp.ServerPort = serverPort

	serverURL := req.Server
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", serverPort)
	}
	normalizedServer := strings.TrimRight(strings.TrimSpace(serverURL), "/")

	if err := writeRemoteAgentConfig(remoteConfigPath, normalizedServer, req.Token); err != nil {
		return nil, err
	}

	killPort(serverPort)

	serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(serverPort), "--credentials-file", credFile)
	serverCmd.Dir = serverHome
	serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
	serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
	serverCmd.Env = append(serverCmd.Env, "HOME="+serverHome)
	serverCmd.Env = append(serverCmd.Env, "AI_CRITIC_NO_OPEN_BROWSER=1")
	if err := serverCmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}
	t.Cleanup(func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(150 * time.Millisecond)
			serverCmd.Process.Kill()
		}
	})

	pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", serverPort)
	if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
		return nil, err
	}
	if err := verifyServerHome(t, normalizedServer, req.Token, serverHome); err != nil {
		return nil, err
	}

	agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
	agentEnv = append(agentEnv, "HOME="+agentHome)

	if req.PrereqSetConfig {
		setExcludes := req.SetConfigExcludePaths
		if len(setExcludes) == 0 {
			setExcludes = req.ExcludePaths
		}
		setArgs := []string{"--server", serverURL, "--token", req.Token, "machine", "backup", "--set-config"}
		for _, ex := range setExcludes {
			setArgs = append(setArgs, "--exclude", ex)
		}
		if req.PrereqSetConfigLargeDirThreshold != "" {
			setArgs = append(setArgs, "--large-dir-threshold", req.PrereqSetConfigLargeDirThreshold)
		}
		t.Logf("prereq set-config argv: %v", setArgs)
		if code, out, errOut, runErr := runAgent(agentBin, setArgs, agentEnv); runErr != nil {
			return nil, runErr
		} else if code != 0 {
			return nil, fmt.Errorf("prereq set-config exit %d:\n%s\n%s", code, out, errOut)
		}
		if len(req.PostPrereqSetConfigExcludes) > 0 {
			if err := appendPostPrereqSetConfigExcludes(t, serverHome, req.PostPrereqSetConfigExcludes); err != nil {
				return nil, err
			}
		}
	}

	if req.PrereqBackup {
		var backupPath string
		if needsCustomPrereqArchive(req) {
			backupPath = filepath.Join(agentHome, "prereq-backup.tar.xz")
			backupArgs := []string{"--server", serverURL, "--token", req.Token, "machine", "backup", "--output", backupPath}
			for _, ex := range req.ExcludePaths {
				backupArgs = append(backupArgs, "--exclude", ex)
			}
			for _, inc := range req.IncludePaths {
				backupArgs = append(backupArgs, "--include", inc)
			}
			if req.SkipGitDirsScan {
				backupArgs = append(backupArgs, "--skip-git-dirs-scan")
			}
			if req.GitDirsScanMaxDepth > 0 {
				backupArgs = append(backupArgs, "--git-dirs-scan-max-depth", strconv.Itoa(req.GitDirsScanMaxDepth))
			}
			t.Logf("prereq backup argv: %v", backupArgs)
			if code, out, errOut, runErr := runAgent(agentBin, backupArgs, agentEnv); runErr != nil {
				return nil, runErr
			} else if code != 0 {
				return nil, fmt.Errorf("prereq backup exit %d:\n%s\n%s", code, out, errOut)
			}
		} else {
			var err error
			backupPath, err = ensureSessionDefaultArchive(t, moduleRoot, serverBin, agentBin, cacheDir, req.Token)
			if err != nil {
				return nil, err
			}
			t.Logf("prereq backup: reusing session archive %s", backupPath)
		}
		req.RestoreArchive = backupPath
		resp.BackupPath = backupPath

		switch req.AfterBackupMutate {
		case "":
		case "modify-bashrc":
			bashrcPath := filepath.Join(serverHome, ".bashrc")
			if err := os.WriteFile(bashrcPath, []byte("mutated after backup\n"), 0644); err != nil {
				return nil, err
			}
			if data, readErr := os.ReadFile(bashrcPath); readErr != nil {
				t.Logf("post-mutation serverHome/.bashrc read error: %v", readErr)
			} else {
				t.Logf("post-mutation serverHome/.bashrc: %q", string(data))
			}
		case "wipe-backup-config":
			writeServerFile(t, serverHome, ".backup/config.json", `{"wiped":true}`+"\n")
		default:
			return nil, fmt.Errorf("unknown AfterBackupMutate %q", req.AfterBackupMutate)
		}
	}

	argv := make([]string, 0, len(req.Args)+16)
	argv = append(argv, "--server", serverURL, "--token", req.Token)
	argv = append(argv, req.Args...)

	if req.RestoreArchive != "" {
		replaced := false
		for i, arg := range argv {
			if arg == "__RESTORE_ARCHIVE__" {
				argv[i] = req.RestoreArchive
				replaced = true
			}
		}
		if !replaced {
			argv = insertRestoreArchive(argv, req.RestoreArchive)
		}
	}

	var subcommandFlags []string
	for _, ex := range req.ExcludePaths {
		subcommandFlags = append(subcommandFlags, "--exclude", ex)
	}
	for _, inc := range req.IncludePaths {
		subcommandFlags = append(subcommandFlags, "--include", inc)
	}
	if req.ShowConfig && !argvContainsFlag(argv, "--show-config") {
		subcommandFlags = append(subcommandFlags, "--show-config")
	}
	if req.ShowMeta && !argvContainsFlag(argv, "--show-meta") {
		subcommandFlags = append(subcommandFlags, "--show-meta")
	}
	if req.SetConfig && !argvContainsFlag(argv, "--set-config") {
		subcommandFlags = append(subcommandFlags, "--set-config")
	}
	if req.SetConfigLargeDirThreshold != "" && !argvContainsFlag(argv, "--large-dir-threshold") {
		subcommandFlags = append(subcommandFlags, "--large-dir-threshold", req.SetConfigLargeDirThreshold)
	}
	if req.SkipGitDirsScan && !argvContainsFlag(argv, "--skip-git-dirs-scan") {
		subcommandFlags = append(subcommandFlags, "--skip-git-dirs-scan")
	}
	if req.GitDirsScanMaxDepth > 0 && !argvContainsFlag(argv, "--git-dirs-scan-max-depth") {
		subcommandFlags = append(subcommandFlags, "--git-dirs-scan-max-depth", strconv.Itoa(req.GitDirsScanMaxDepth))
	}
	if len(subcommandFlags) > 0 {
		argv = insertSubcommandFlags(argv, subcommandFlags...)
	}

	if req.OutputPath != "" {
		out := req.OutputPath
		if !filepath.IsAbs(out) {
			out = filepath.Join(agentHome, out)
		}
		absOut, err := filepath.Abs(out)
		if err != nil {
			return nil, err
		}
		req.OutputPath = absOut
		for i, arg := range argv {
			if arg == "__OUTPUT_PATH__" {
				argv[i] = absOut
			}
		}
	}

	if req.DryRunThenArchive {
		return runDryRunThenArchive(t, req, resp, agentBin, agentEnv, serverURL)
	}

	t.Logf("remote-agent argv: %v", argv)

	exitCode, stdout, stderr, runErr := runAgent(agentBin, argv, agentEnv)
	if runErr != nil {
		return nil, runErr
	}

	resp.ExitCode = exitCode
	resp.Stdout = stdout
	resp.Stderr = stderr
	resp.Combined = strings.TrimSpace(resp.Stdout + "\n" + resp.Stderr)

	if resp.BackupPath == "" && req.OutputPath != "" {
		resp.BackupPath = req.OutputPath
	}

	if req.FollowUpShowConfig {
		showArgv := []string{"--server", serverURL, "--token", req.Token, "machine", "backup", "--show-config"}
		t.Logf("follow-up show-config argv: %v", showArgv)
		code, stdout, stderr, runErr := runAgent(agentBin, showArgv, agentEnv)
		if runErr != nil {
			return nil, runErr
		}
		if code != 0 {
			return nil, fmt.Errorf("follow-up show-config exit %d:\n%s\n%s", code, stdout, stderr)
		}
		resp.FollowUpStdout = stdout
	}

	return resp, nil
}

func runAgent(bin string, argv, env []string) (int, string, string, error) {
	cmd := exec.Command(bin, argv...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, "", "", runErr
		}
	}
	return exitCode, outBuf.String(), errBuf.String(), nil
}

type remoteAgentConfigFile struct {
	Default string            `json:"default,omitempty"`
	Domains []domainConfigRow `json:"domains"`
}

type domainConfigRow struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

func writeRemoteAgentConfig(path, server, token string) error {
	cfg := remoteAgentConfigFile{
		Default: server,
		Domains: []domainConfigRow{{Server: server, Token: token}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
		}
		dir = parent
	}
}

func portBaseFromTestName(name string) int {
	hash := 0
	for _, c := range name {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return 26000 + (hash % 1000)
}

func pickFreePort(base int) int {
	for port := base; port < base+200; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	panic(fmt.Sprintf("no free port near %d", base))
}

func killPort(port int) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	for _, pidStr := range strings.Fields(strings.TrimSpace(string(out))) {
		_ = exec.Command("kill", "-9", pidStr).Run()
	}
}

func normalizeAbsPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

func verifyServerHome(t *testing.T, serverURL, token, wantHome string) error {
	want, err := normalizeAbsPath(wantHome)
	if err != nil {
		return fmt.Errorf("resolve harness serverHome: %w", err)
	}
	backupURL := strings.TrimRight(strings.TrimSpace(serverURL), "/") + "/api/remote-agent/machine/backup"
	body := `{"dry_run":true,"exclude":[],"include":[]}`
	req, err := http.NewRequest(http.MethodPost, backupURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build verify-home request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify server HOME: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify-home response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify server HOME status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var plan struct {
		Home string `json:"home"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return fmt.Errorf("decode backup plan for HOME verify: %w", err)
	}
	got, err := normalizeAbsPath(plan.Home)
	if err != nil {
		return fmt.Errorf("resolve server-reported HOME %q: %w", plan.Home, err)
	}
	if got != want {
		return fmt.Errorf(
			"server HOME mismatch on %s: server reports %q (normalized %q) but harness serverHome is %q (normalized %q); stale process may still be bound to the port",
			backupURL, plan.Home, got, wantHome, want,
		)
	}
	t.Logf("verified server HOME=%s", got)
	return nil
}

func waitHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func argvContainsFlag(argv []string, flag string) bool {
	for _, arg := range argv {
		if arg == flag {
			return true
		}
	}
	return false
}

func subcommandFlagInsertAt(argv []string) int {
	for i, arg := range argv {
		if arg == "machine" && i+1 < len(argv) {
			insertAt := i + 2
			if argv[i+1] == "restore" && insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
				insertAt++
			}
			for insertAt < len(argv) && strings.HasPrefix(argv[insertAt], "-") {
				insertAt++
				if insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
					insertAt++
				}
			}
			return insertAt
		}
	}
	return len(argv)
}

func insertRestoreArchive(argv []string, archive string) []string {
	for i, arg := range argv {
		if arg != "machine" || i+1 >= len(argv) || argv[i+1] != "restore" {
			continue
		}
		insertAt := i + 2
		if insertAt < len(argv) && !strings.HasPrefix(argv[insertAt], "-") {
			return argv
		}
		rest := append([]string{archive}, argv[insertAt:]...)
		return append(append([]string{}, argv[:insertAt]...), rest...)
	}
	return argv
}

func insertSubcommandFlags(argv []string, flags ...string) []string {
	insertAt := subcommandFlagInsertAt(argv)
	out := make([]string, 0, len(argv)+len(flags))
	out = append(out, argv[:insertAt]...)
	out = append(out, flags...)
	out = append(out, argv[insertAt:]...)
	return out
}

func stripEnvPrefix(env []string, prefix string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func ensureSessionDefaultArchive(t *testing.T, moduleRoot, serverBin, agentBin, cacheDir, token string) (string, error) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(cacheDir, "default-prereq.tar.xz")
	lock := filepath.Join(cacheDir, "default-backup.lock")
	err := withFileLock(t, lock, func() error {
		if archiveHasXZMagicFile(archive) {
			return nil
		}
		seedHome, err := os.MkdirTemp("", "machine-backup-session-seed-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(seedHome)
		if err := seedServerHome(t, seedHome, false); err != nil {
			return err
		}
		credDir := filepath.Join(seedHome, ".ai-critic")
		if err := os.MkdirAll(credDir, 0755); err != nil {
			return err
		}
		credFile := filepath.Join(credDir, "server-credentials")
		if err := os.WriteFile(credFile, []byte(token+"\n"), 0600); err != nil {
			return err
		}
		agentHome, err := os.MkdirTemp("", "machine-backup-session-agent-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(agentHome)
		port := pickFreePort(27000 + portBaseFromTestName(DOCTEST_SESSION_ID)%500)
		serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)
		killPort(port)
		serverCmd := exec.Command(serverBin, "--port", strconv.Itoa(port), "--credentials-file", credFile)
		serverCmd.Dir = seedHome
		serverCmd.Env = stripEnvPrefix(os.Environ(), "HOME=")
		serverCmd.Env = stripEnvPrefix(serverCmd.Env, lib.EnvAI_CRITIC_HOME+"=")
		serverCmd.Env = append(serverCmd.Env, "HOME="+seedHome, "AI_CRITIC_NO_OPEN_BROWSER=1")
		if err := serverCmd.Start(); err != nil {
			return fmt.Errorf("start session seed server: %w", err)
		}
		defer func() {
			if serverCmd.Process != nil {
				serverCmd.Process.Signal(syscall.SIGTERM)
				time.Sleep(100 * time.Millisecond)
				serverCmd.Process.Kill()
			}
		}()
		pingURL := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
		if err := waitHTTPReady(pingURL, 30*time.Second); err != nil {
			return err
		}
		if err := verifyServerHome(t, serverURL, token, seedHome); err != nil {
			return err
		}
		agentEnv := stripEnvPrefix(os.Environ(), "HOME=")
		agentEnv = append(agentEnv, "HOME="+agentHome)
		backupArgs := []string{"--server", serverURL, "--token", token, "machine", "backup", "--output", archive}
		t.Logf("session default backup argv: %v", backupArgs)
		if code, out, errOut, runErr := runAgent(agentBin, backupArgs, agentEnv); runErr != nil {
			return runErr
		} else if code != 0 {
			return fmt.Errorf("session default backup exit %d:\n%s\n%s", code, out, errOut)
		}
		if !archiveHasXZMagicFile(archive) {
			return fmt.Errorf("session default archive missing xz magic: %s", archive)
		}
		t.Logf("session default archive written: %s", archive)
		return nil
	})
	if err != nil {
		return "", err
	}
	return archive, nil
}
```