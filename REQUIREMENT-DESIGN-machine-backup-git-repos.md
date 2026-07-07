# Machine Backup: Git Repo / Worktree Meta

## Summary

During `machine backup` (dry-run and real backup), discover git repositories under
included top-level dot-dirs in server `HOME`, capture branch/commit/status/worktree
metadata, and:

1. Emit a **`GIT REPOS`** section in the dry-run summary (verbatim `log` frames).
2. Write **`.backup/git-repo-worktrees.json`** into real backup archives at pack time.
3. Expose the JSON via **`restore --show-meta`** (automatic via `ReadArchiveMeta`).

Restore apply does **not** re-scan or write git meta to live `~/.backup/`.

## Locked design (user-approved)

| Item | Decision |
|------|----------|
| `commit_sha` | 7-char short (`git rev-parse --short=7 HEAD`) |
| `status` | `"clean"` or `"dirty (N modified, M added, …)"` from `git status --porcelain` counts |
| No repos | `GIT REPOS: (none)` |
| `--skip-git-dirs-scan` | Skip scan; summary shows `GIT REPOS: (skipped)`; no JSON in archive |
| `--git-dirs-scan-max-depth` | Default `0` = unlimited; `N > 0` caps `scan_repo.MaxDepth` |
| `--show-meta` | Include `git-repo-worktrees.json` (like `installed.json`) |

User-facing stdout ends with `\n` after the last content line.

## Data model

### JSON file: `.backup/git-repo-worktrees.json`

```json
{
  "version": "1.0",
  "captured_at": "2026-07-07T12:00:00Z",
  "repos": [
    {
      "path": ".wrk/my-project",
      "branch": "main",
      "commit_sha": "abc1234",
      "commit_msg": "fix backup meta",
      "status": "clean",
      "worktrees": [
        {
          "path": ".wrk/my-project-wt",
          "branch": "feature/foo",
          "commit_sha": "def5678",
          "commit_msg": "wip feature",
          "status": "dirty (1 modified)"
        }
      ]
    }
  ]
}
```

- `path` values are **relative to server HOME** (same style as backup plan paths).
- Main repos only in top-level `repos[]`; linked worktrees nested under `worktrees[]`.
- Worktree checkout rows discovered as separate `scan_repo` repos are folded into parent
  `worktrees[]` (not duplicate top-level `repos[]` entries).
- Sort `repos[]` by `path` ascending; sort `worktrees[]` by `path` ascending.

### Go types (implementer guidance)

```go
const metaGitReposName = "git-repo-worktrees.json"

type GitRepoWorktreesSnapshot struct {
    Version    string         `json:"version"`     // "1.0"
    CapturedAt time.Time      `json:"captured_at"`
    Repos      []GitRepoEntry `json:"repos"`
}

type GitRepoEntry struct {
    Path       string            `json:"path"`
    Branch     string            `json:"branch"`
    CommitSHA  string            `json:"commit_sha"`
    CommitMsg  string            `json:"commit_msg"`
    Status     string            `json:"status"`
    Worktrees  []GitWorktreeEntry `json:"worktrees,omitempty"`
}

type GitWorktreeEntry struct {
    Path      string `json:"path"`
    Branch    string `json:"branch"`
    CommitSHA string `json:"commit_sha"`
    CommitMsg string `json:"commit_msg"`
    Status    string `json:"status"`
}
```

Add to `MachineBackupPlan` (optional, for `done` frame / tests):

```go
GitRepos *GitRepoWorktreesSnapshot `json:"git_repos,omitempty"`
```

### API / CLI flags

**CLI** (`remote-agent machine backup` only):

```
--skip-git-dirs-scan
--git-dirs-scan-max-depth N
```

**BackupStreamRequest / BackupRequest** gain:

```go
SkipGitDirsScan      bool `json:"skip_git_dirs_scan,omitempty"`
GitDirsScanMaxDepth  int  `json:"git_dirs_scan_max_depth,omitempty"` // 0 = unlimited
```

Wire through: CLI → JSON body (dry-run stream) and archive endpoint query/body.

## Scan behavior

- Use `github.com/xhd2015/dot-pkgs/go-pkgs/git/scan_repo`.
- **Roots**: absolute paths of each **included** top-level dot-dir from the backup plan
  (`DirStat.Path` under `HOME`). Do not scan excluded trees.
- **IgnoreDirs**: derive from backup `ExclusionRules` (excluded path prefixes under each root).
- **ListWorktrees**: `true` on main repos.
- **MaxDepth**: `GitDirsScanMaxDepth` (0 = unlimited).
- Per-repo enrichment via git subprocess in repo checkout:
  - `git rev-parse --abbrev-ref HEAD` → branch (or `HEAD` when detached)
  - `git rev-parse --short=7 HEAD` → commit_sha
  - `git log -1 --format=%s` → commit_msg (single line; escape/sanitize for JSON)
  - `git status --porcelain` → status string with counts

### Status string format

| Condition | `status` |
|-----------|----------|
| Empty porcelain | `"clean"` |
| Dirty | `"dirty (N modified, M added, K deleted, U untracked, …)"` — only non-zero counts, human labels |

Use standard porcelain first-column / second-column semantics:
- modified (working tree and/or index)
- added (staged new)
- deleted
- untracked (`??`)
- renamed, copied, etc. as applicable

## Dry-run summary: `GIT REPOS` section

Insert after `EXCLUDED` / before `TOTAL:` (or after LARGE DIR DETAIL if present).

**With repos:**

```
  GIT REPOS:
    .wrk/my-project
      branch main  abc1234  clean
      fix backup meta
      worktree .wrk/my-project-wt
        branch feature/foo  def5678  dirty (1 modified)
        wip feature
```

**No repos:**

```
  GIT REPOS: (none)
```

**Skipped (`--skip-git-dirs-scan`):**

```
  GIT REPOS: (skipped)
```

Formatting rules:
- Two-space indent for section header content (match existing summary style).
- Repo path on its own indented line; status line: `branch <name>  <sha>  <status>`.
- Commit message on next line (no prefix).
- Worktrees prefixed with `worktree <rel-path>` then nested branch/sha/status/msg lines
  (extra indent).

## Archive / restore

- **Real backup**: inject `.backup/git-repo-worktrees.json` in `writeBackupMeta` (or sibling
  helper) alongside `config.json`, `installed.json`, `ENV`.
- **Skipped scan**: omit file from archive entirely.
- **`isBackupMetaSnapshot`**: add `git-repo-worktrees.json` so restore apply skips it
  (snapshot only, not written to live `~/.backup/`).
- **`ReadArchiveMeta`**: no change needed — new file included automatically.
- **`restore/show-meta` doctest**: extend to assert `=== .backup/git-repo-worktrees.json ===`
  when prereq backup had git fixtures.

## Test strategy

Extend existing tree `tests/remote-agent-machine-backup` (do not create a new top-level feature dir).

### Harness extensions

Add `Request` fields:

```go
// SeedGitRepos adds git fixtures under a dot-dir (e.g. .wrk-test).
SeedGitRepos bool

// SeedGitReposWorktree also adds a linked worktree + dirty file in worktree.
SeedGitReposWorktree bool

// SkipGitDirsScan sets --skip-git-dirs-scan on backup invocation.
SkipGitDirsScan bool

// GitDirsScanMaxDepth sets --git-dirs-scan-max-depth (0 = omit flag).
GitDirsScanMaxDepth int

// ShowMeta for restore leaves (existing field).
```

Add root `SETUP.md` helpers (skip leaf when `git` not on PATH, same pattern as
`tests/remote-agent-machine-analyse-files`):

- `gitInitRepo(t, dir)` — init + initial commit
- `gitWorktreeAdd(t, mainDir, wtDir, branch)`
- `seedGitReposFixture(t, home)` — e.g. `.wrk-test/main/` git repo with commit message
  `backup git fixture`
- `seedGitReposWorktreeFixture(t, home)` — above + worktree at `.wrk-test/feature-wt`
  with one modified tracked file (dirty status)

Ensure seeded dot-dir is **included** in backup (not built-in excluded). Use a dedicated
top-level name like `.wrk-test` (not `.wrk` if ambiguous).

For max-depth leaf: nest repo at depth exceeding limit (e.g. `.wrk-test/a/b/c/deep-repo`)
with `--git-dirs-scan-max-depth 2` → `(none)`.

### New leaves

| Leaf | Purpose |
|------|---------|
| `backup/git-repos-summary` | Dry-run: `GIT REPOS` lists main repo (branch, short sha, clean, commit msg) |
| `backup/git-repos-worktree` | Dry-run: nested worktree block + dirty status with count |
| `backup/git-repos-none` | Default seed (no git) → `GIT REPOS: (none)` |
| `backup/git-repos-skipped` | `--skip-git-dirs-scan` → `GIT REPOS: (skipped)`; no JSON in archive |
| `backup/git-repos-max-depth` | Deep repo beyond max depth → `(none)` |
| `backup/git-repos-archive` | Real backup archive contains valid `git-repo-worktrees.json` |
| `restore/show-meta-git-repos` | Prereq backup with git → `--show-meta` prints git JSON section |

### Updates

| Leaf | Change |
|------|--------|
| `restore/show-meta` | Clarify it still requires installed.json + ENV; git leaf is separate |
| Root `DOCTEST.md` | DSN + decision tree + test index entries for new leaves and flags |

### Assertions guidance

- Parse summary substring after `GIT REPOS` for dry-run leaves.
- For archive leaf: `tarXZExtractFile(..., ".backup/git-repo-worktrees.json")`, unmarshal,
  assert `version`, `captured_at`, repo path, 7-char sha, status.
- `git-repos-skipped`: archive member list must **not** contain
  `.backup/git-repo-worktrees.json`.
- `git-repos-worktree`: combined output contains `worktree` line and `dirty (` with count.

## Unit tests (implementer, outside doctest)

- `formatGitStatusFromPorcelain` count aggregation
- `relPathFromHome` / worktree nesting builder
- Summary `GIT REPOS` formatter

## Verification

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```

## Approved

User `/doctest-tdd go ahead` after followup locked all clarification items.