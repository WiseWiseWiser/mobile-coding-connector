# Consolidate wrkcli Git Enrichment into dot-pkgs (PR-5)

## Summary

Migrate duplicated git subprocess + porcelain parse + status formatting from
`wrkcli/status.go` (and shared helpers used by `projects_gather.go`) into the
existing dot-pkgs packages from PR-1–4 (`git/cmd`, `git/status`, `git/checkout`).

**Path:** `external/dot-pkgs-master-2026-07-07/go-pkgs/`

User `/doctest-tdd go ahead consolidating` continues the consolidation plan.
PR-6 (`git/worktree` overlap) remains out of scope.

## Locked decisions

| Item | Decision |
|------|----------|
| wrk status taxonomy | Four buckets: `added`, `changed`, `renamed`, `deleted` |
| `??` in wrk parse | Counts as **added** (not `untracked`) |
| `M` / default porcelain | Counts as **changed** (not `modified`) |
| wrk porcelain fetch | `git status --porcelain --untracked-files=no` for normal status blocks |
| wrk format string | `clean` or `dirty (N added, N changed, N renamed, N deleted)` — always four segments when dirty |
| Color / ANSI | Stays in `wrkcli` (wrap plain text from `git/status`) |
| `checkout.Enrich` | Add `StatusStyle` + `PorcelainUntracked` options; wrk uses `FormatWrk` + `--untracked-files=no` |
| `git/cmd` | Replace `wrkcli/gitOutput`, `gitCombinedOutput` for read-only git calls |
| `gitexec.go` | Keep `gitCommand`/`exec.Cmd` for mutating ops (worktree add, fetch with CombinedOutput); route **read** helpers through `git/cmd` |
| Backup regressions | No behavior change in ai-critic machine backup git-repos leaves |
| wrk regressions | All existing `cmd/wrk/tests/status/**` leaves must pass unchanged |

## API extensions (`git/status`)

```go
// WrkCounts is the wrk four-bucket view (distinct from backup Counts labels).
type WrkCounts struct {
    Added, Changed, Renamed, Deleted int
}

// ParsePorcelainWrk applies wrk taxonomy (?? → added; M/default → changed).
func ParsePorcelainWrk(porcelain string) WrkCounts

// FormatWrk renders wrk --status Status: value (no ANSI).
func FormatWrk(counts WrkCounts) string
```

Also wire `Format(Counts, FormatWrk)` to convert via an internal mapping **or**
deprecate in favor of `FormatWrk(ParsePorcelainWrk(...))` only — implementer
chooses minimal surface; doctests target `FormatWrk` + `ParsePorcelainWrk` directly.

`FormatWrk` examples (locked):

| WrkCounts | Output |
|-----------|--------|
| all zero | `clean` |
| `{Added:1, Changed:1, Renamed:1, Deleted:1}` | `dirty (1 added, 1 changed, 1 renamed, 1 deleted)` |
| `{Changed:1}` | `dirty (0 added, 1 changed, 0 renamed, 0 deleted)` |

## API extensions (`git/checkout`)

```go
type Options struct {
    ShortSHALength      int
    StatusStyle         status.FormatStyle // FormatBackup (default) | FormatWrk
    PorcelainUntracked  bool               // default true; wrk sets false
}
```

When `StatusStyle == FormatWrk`, enrichment uses `ParsePorcelainWrk` +
`FormatWrk` for `Meta.Status`.

## wrkcli refactor targets

### `status.go`

- `printStatusBlock` / `printAppendedLinkedBlock`: use `checkout.Enrich` with wrk
  options for branch/commit/status; keep `Master:` / `Remote:` / color in wrkcli.
- Delete: `statusCounts` type, `parseStatusCounts`, `countStatusLine`,
  `gitStatusCounts`, non-color core of `formatStatusCounts` (keep color wrapper
  calling `status.FormatWrk`).
- Route `gitOutput` read paths through `git/cmd.Run` (context.Background()).

### `projects_gather.go`

- Replace `statusCounts` usage with `status.WrkCounts` / `ParsePorcelainWrk`.
- `gitProjectStatusCountsWithSkip`, `parseProjectStatusCounts`: delegate parse to
  `status.ParsePorcelainWrk` with skip-untracked path filter preserved.
- `gitWorktreeIsClean`: use `ParsePorcelainWrk` zero check.
- `formatStatusCounts` callers unchanged at stdout level.

### Keep in wrkcli

- `gitCommand` / `gitCommandDir` / fetch helpers for mutating git
- `formatStatusCounts` color wrapper
- `masterBriefForRepo`, remote compare, worktree discovery

## Test strategy

### New dot-pkgs doctest leaves

Extend `git/status/tests/`:

| Leaf | Description |
|------|-------------|
| `parse/wrk-mixed` | Porcelain lines map to wrk buckets (`??`→added, `M`→changed, `R`, `D`) |
| `format/wrk-clean` | Zero WrkCounts → `"clean"` |
| `format/wrk-dirty` | `{1,1,1,1}` → full four-segment dirty line |
| `format/wrk-partial` | `{Changed:1}` → `dirty (0 added, 1 changed, 0 renamed, 0 deleted)` |

Extend root `DOCTEST.md` tree index + `Run` to support `parse-wrk` and
`format-wrk` ops (or reuse `parse`/`format` with `Style: FormatWrk`).

Optional `git/checkout/tests/enrich/wrk-style` — enrich clean repo with
`StatusStyle: FormatWrk` yields `Status: clean`.

### Regression (no new leaves)

```sh
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/cmd/wrk/tests/status/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/status/tests/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/checkout/tests/...
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
```

## Verification

```sh
cd external/dot-pkgs-master-2026-07-07
doctest vet ./go-pkgs/git/status/tests/
doctest test ./go-pkgs/git/status/tests/...
go test ./go-pkgs/wrkcli/... -count=1

cd <ai-critic-root>
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-...
```

## Approved

User `/doctest-tdd go ahead consolidating` on PR-5 wrkcli migration.