# Consolidate Git Enrichment into dot-pkgs (PR-1–4)

## Summary

Move duplicated git subprocess + checkout enrichment + repo tree building from
ai-critic `machinebackup/git_repos.go` (and align with wrkcli patterns) into new
dot-pkgs packages. ai-critic keeps only backup JSON mapping + summary layout.

**Path:** `external/dot-pkgs-master-2026-07-07/go-pkgs/git/`

User `/doctest-tdd go ahead` after followup refactor plan. wrkcli migration (PR-5)
**out of scope** for this cycle.

## Locked decisions

| Item | Decision |
|------|----------|
| Status taxonomy | Canonical `status.Counts`; `FormatStyle` = `Backup` (machine backup) and `Wrk` (reserved) |
| `reposnapshot` paths | `rel(absPath string) string` callback from caller |
| JSON version | Stay backup `1.0`; optional `error` unchanged |
| wrkcli | Defer PR-5; no wrk output changes this cycle |

## New packages

### `git/cmd`

```go
func Run(ctx context.Context, dir string, args ...string) (string, error)
func RunOptional(ctx context.Context, dir string, args ...string) (string, bool, error)
func RunCombined(ctx context.Context, dir string, args ...string) (string, error)
```

Sets `GIT_OPTIONAL_LOCKS=0` when appropriate. Normalize errors (one-line gist).

### `git/status`

```go
type Counts struct {
    Modified, Added, Deleted, Untracked, Renamed, Copied, Unmerged int
}
func ParsePorcelain(string) Counts
type FormatStyle int
const FormatBackup FormatStyle = iota  // "dirty (N modified, M added, ...)"
func Format(Counts, FormatStyle) string // clean | dirty (...)
```

Backup style matches existing `machinebackup` output (`modified` not `changed`;
`??` → `untracked`).

### `git/checkout`

```go
type Meta struct {
    Branch, CommitSHA, CommitMsg, Status, Error string
}
type Options struct { ShortSHALength int } // default 7
func Enrich(ctx context.Context, repoPath string, opts Options) Meta
```

Durable stepwise enrichment (branch → sha → msg → status). Partial fields + `Error`.
Unborn HEAD → `no commits (HEAD unborn)`. Never returns error to caller.

### `git/reposnapshot`

```go
type Node struct {
    Path      string
    Checkout  checkout.Meta
    Worktrees []Node
    Error     string // merged scan_repo.Repo.Error
}
type Snapshot struct {
    Nodes      []Node
    RootErrors []RootErrorEntry // Path + Error (rel paths)
}
func Build(result scan_repo.Result, rel func(abs string) string) Snapshot
```

Owns main+worktree nesting (today's `collectWorktreePaths` + `buildGitReposSnapshot`).

## ai-critic changes (PR-4)

`server/machinebackup/git_repos.go`:

- `ScanGitRepos`: `scan_repo.Scan` → `reposnapshot.Build` → map to `GitRepoWorktreesSnapshot`
- Delete: `enrichGitCheckout`, `gitOutput`, porcelain parsers, `collectWorktreePaths`, most of `buildGitReposSnapshot`
- Keep: `formatGitReposSummaryLines`, `marshalGitReposSnapshot`, backup-specific types, ignore roots wiring
- **No** `exec.Command("git", ...)` in `machinebackup` after refactor

Behavior unchanged: existing 8 git-repos doctests + `git-repos-empty-repo` must pass.

## `scan_repo` refactor

- Replace private `gitOutput`/`gitOptionalOutput` with `git/cmd`
- No API break beyond existing `Result` type

## Test strategy

### dot-pkgs doctest trees (new)

| Tree | Leaves |
|------|--------|
| `git/cmd/tests` | `run/success`, `run/missing-repo` (or skip if git unavailable) |
| `git/status/tests` | `parse/clean`, `parse/mixed`, `format/backup-dirty`, `format/backup-clean` |
| `git/checkout/tests` | `enrich/clean-repo`, `enrich/empty-repo`, `enrich/partial` (optional) |
| `git/reposnapshot/tests` | `build/main-and-worktree`, `build/root-error` |

Follow `git/scan_repo/tests` harness style (temp dirs, `git` on PATH, skip when missing).

### ai-critic regression

No new leaves. Re-run all `tests/remote-agent-machine-backup/backup/git-repos-*` and `restore/show-meta-git-repos`.

### Unit tests

Move `formatGitStatusFromPorcelain` tests from `git_repos_test.go` to `git/status` or delete if covered by doctests.

## Verification

```sh
doctest vet ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/cmd/tests
doctest vet ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/status/tests
doctest vet ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/checkout/tests
doctest vet ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/reposnapshot/tests
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/.../tests/...

doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests/...

doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
# ... all git-repos leaves

go test ./server/machinebackup/... -count=1
```

## Approved

User `/doctest-tdd go ahead` on consolidation plan.