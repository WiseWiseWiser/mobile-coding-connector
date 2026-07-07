# Implement: wrkcli Git Consolidation (PR-5)

## Context

Migrate wrkcli git duplication into dot-pkgs packages. Design:
`REQUIREMENT-DESIGN-wrkcli-git-consolidate.md`.

PR-1–4 already delivered `git/cmd`, `git/status` (backup), `git/checkout`,
`git/reposnapshot`. This cycle adds wrk taxonomy and rewires wrkcli.

## Tests sealed — do not modify

In `external/dot-pkgs-master-2026-07-07` (git-sealed):

- `go-pkgs/git/status/tests/` — includes new `parse/wrk-mixed`, `format/wrk-*`
- `go-pkgs/git/checkout/tests/` — includes new `enrich/wrk-style`

ai-critic `tests/remote-agent-machine-backup/` — do not modify.

## Implementation order

### 1. `git/status` — wrk API

- Add `WrkCounts`, `ParsePorcelainWrk`, `FormatWrk` function
- Resolve `FormatWrk` name collision: const `FormatWrk` vs function — rename
  const to e.g. `StyleWrk` or use separate `FormatWrk(counts)` function and
  keep `FormatStyle` enum; doctests call `status.FormatWrk(WrkCounts)`
- Wrk parse rules: `??`→added, `R`→renamed, `A`→added, `D`→deleted, else→changed
- `FormatWrk`: `clean` or `dirty (N added, N changed, N renamed, N deleted)`

### 2. `git/checkout` — wrk options

- Extend `Options` with `StatusStyle` and `PorcelainUntracked` (default true)
- When `StatusStyle == FormatWrk` (or StyleWrk): porcelain via
  `--untracked-files=no` when `PorcelainUntracked == false`; status via
  `ParsePorcelainWrk` + `FormatWrk`
- Default behavior unchanged for backup

### 3. `wrkcli/status.go`

- Use `checkout.Enrich` with wrk options in `printStatusBlock` /
  `printAppendedLinkedBlock` where appropriate
- Replace `gitStatusCounts`/`parseStatusCounts` with `status.ParsePorcelainWrk`
- `formatStatusCounts`: plain text from `status.FormatWrk`; keep ANSI wrapper
- Route read-only git via `git/cmd.Run` instead of `gitOutput`
- Delete dead helpers: `statusCounts`, `countStatusLine`, `parseStatusCounts`,
  `gitStatusCounts` (if fully replaced)

### 4. `wrkcli/projects_gather.go`

- Use `status.WrkCounts` / `ParsePorcelainWrk` / `FormatWrk`
- Preserve `parseProjectStatusCounts` skip-untracked path logic
- `gitWorktreeIsClean`: zero `WrkCounts` check

### 5. `git/cmd` in wrkcli

- Replace `gitOutput`/`gitOutputNoOptionalLocks` read paths with `git/cmd.Run`
- Keep `gitexec.go` for mutating commands (fetch, worktree add)

## Verify

```sh
cd external/dot-pkgs-master-2026-07-07

doctest vet ./go-pkgs/git/status/tests/
doctest test ./go-pkgs/git/status/tests/...

doctest vet ./go-pkgs/git/checkout/tests/
doctest test ./go-pkgs/git/checkout/tests/...

doctest test ./go-pkgs/cmd/wrk/tests/status/...

go test ./go-pkgs/wrkcli/... -count=1
go test ./go-pkgs/git/... -count=1

cd <ai-critic-root>
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
```

All tests must be GREEN. Do not modify sealed doctest trees.