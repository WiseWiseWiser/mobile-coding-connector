# Git Scan Error Durability (scan_repo + machine backup)

## Summary

Git repo discovery/enrichment must **never abort** backup or dry-run. Per-repo,
per-worktree, and per-root failures are recorded as optional `error` on the
entry. Partial field success is allowed alongside `error`.

Fixes live failure: `enrich main repo /root/.openclaw/workspace: git rev-parse HEAD`
(empty repo, no commits).

## Locked design

| Item | Decision |
|------|----------|
| JSON version | Stay **1.0**; add optional `error` field |
| Per-root scan fail | Synthetic entry `{"path": "<dot-dir>", "error": "scan failed: ..."}` |
| Partial enrichment | Keep fields gathered; append `error` for failed steps (`"; "` joined) |
| scan_repo | Must be error-durable in `external/dot-pkgs-master-2026-07-07` |
| Backup/dry-run | Never fail due to git meta scan |

User-facing stdout ends with `\n`.

## Data model

### ai-critic (`GitRepoEntry`, `GitWorktreeEntry`)

```go
Error string `json:"error,omitempty"`
```

### dot-pkgs (`scan_repo`)

```go
type Repo struct {
    // existing fields...
    Error string `json:"error,omitempty"`
}

type RootError struct {
    Root  string
    Error string
}

type Result struct {
    Repos      []Repo
    RootErrors []RootError
}

// Scan returns (Result, error). error is fatal only:
// empty roots, ctx cancelled, buildIgnoreConfig failure.
func Scan(ctx context.Context, opts Options) (Result, error)
```

**Behavior changes:**

- All roots attempted; `validateRoot` fail → `RootErrors`, continue.
- `walkRoot` fail → `RootErrors` for that root, continue.
- `resolveGitDir` fail at path → `Repo{Path, Error}`, continue walk.
- `enrichRepo` fail → `repo.Error` set, repo still returned.
- `OnRepo` error → attach to `repo.Error`, continue (do not abort walk).

**Breaking test updates (scan_repo):**

| Leaf | Old | New |
|------|-----|-----|
| `scan/missing-root-error` | `err != nil` | `err == nil`, one `RootError` for missing path |
| `scan/not-a-directory-error` | `err != nil` | `err == nil`, one `RootError` |

When **multiple roots**: one bad + one good → good repos returned + `RootError` for bad.

### ai-critic enrichment (`enrichGitCheckout`)

Stepwise; on failure record error and stop further steps for that entry:

1. branch (`rev-parse --abbrev-ref HEAD`)
2. sha (`rev-parse --short=7 HEAD`)
3. msg (`log -1 --format=%s`)
4. status (`status --porcelain`)

Normalize common cases:
- no commits → `no commits (HEAD unborn)`

### Summary display

```
    .openclaw/workspace
      error: no commits (HEAD unborn)
    .wrk-test/main
      branch main  abc1234  clean
      backup git fixture
```

Partial:
```
    .wrk-test/feature-wt
      branch feature/foo  abc1234
      error: git status --porcelain: permission denied
```

Root synthetic:
```
    .bad-root
      error: scan failed: root does not exist
```

## Test strategy

### A. `tests/remote-agent-machine-backup` (extend existing tree)

| Leaf | Purpose |
|------|---------|
| `backup/git-repos-empty-repo` | `git init` only under `.wrk-test/empty` → entry with `error`, exit 0, dry-run completes |
| Update existing git-repos leaves | Still pass; no regression |

Harness: `SeedGitReposEmpty` — init repo, no commit, no add.

### B. `external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests`

| Leaf | Purpose |
|------|---------|
| `scan/root-failure-isolated` | Two roots: valid repo + missing path → 1 repo + 1 RootError, `err == nil` |
| Update `scan/missing-root-error` | Expect RootError not fatal err |
| Update `scan/not-a-directory-error` | Expect RootError not fatal err |
| `enrich-worktrees/list-error-continues` | Optional: if hard to fixture, unit test in implementer |

Update `DOCTEST.md` Response + Run:

```go
type Response struct {
    Repos      []scan_repo.Repo
    RootErrors []scan_repo.RootError
    // ...
}

// Run calls Scan → populate Repos + RootErrors
```

## Implementation notes (for implementer phase)

- Update **all** `scan_repo.Scan` callers in ai-critic and dot-pkgs.
- `ScanGitRepos` maps `RootErrors` → synthetic `GitRepoEntry` with `path` = rel root.
- Map `repo.Error` from scan_repo onto entry if enrichment not attempted or merged.
- `BuildPlan` / `WriteArchive`: git scan errors never propagate as Go error.

## Verification

```sh
# dot-pkgs (from repo root, module replace active)
doctest vet ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests/...

# ai-critic
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-...
go test ./server/machinebackup/... -count=1
```

## Approved

User `/doctest-tdd go ahead fix all` after followup locked all items.