# Implement: Git Scan Error Durability

## Context

`machine backup --dry-run` fails when any discovered git repo cannot be enriched
(e.g. empty repo at `.openclaw/workspace` with no commits). User requires durable
scan: record `error` per entry, allow partial fields, never abort backup.

Design: `REQUIREMENT-DESIGN-git-scan-error-durable.md`

## Tests are sealed — do not modify

- `tests/remote-agent-machine-backup/`
- `external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests/`

## Part 1: dot-pkgs `scan_repo`

Path: `external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/`

1. Add `Repo.Error`, `RootError`, `Result` types.
2. Change `Scan` to return `(Result, error)` — fatal only: empty roots, ctx cancel, `buildIgnoreConfig`.
3. Per-root: `validateRoot` fail → `RootErrors`, continue other roots.
4. `walkRoot` fail → `RootErrors`, continue.
5. `resolveGitDir` fail → emit `Repo{Path, Error}`, continue walk.
6. `enrichRepo` fail → set `repo.Error`, still return repo.
7. `OnRepo` error → non-fatal, attach error, continue.
8. Update **all** callers of `Scan` in dot-pkgs module (grep the module).

## Part 2: ai-critic `machinebackup`

Path: `server/machinebackup/`

1. Add `Error string` to `GitRepoEntry` and `GitWorktreeEntry`.
2. Rewrite `enrichGitCheckout` for stepwise partial enrichment + `error` field.
3. `buildGitReposSnapshot`: never abort on per-repo/worktree enrichment failure.
4. `ScanGitRepos`: use `scan_repo.Result`; map `RootErrors` → synthetic entries;
   never return Go error for scan/enrich failures.
5. `formatGitReposSummaryLines`: print `error:` lines; partial fields when present.
6. `BuildPlan` / `WriteArchive`: git scan must not fail backup.

Normalize: `no commits (HEAD unborn)` for empty repos.

## Verify

```sh
# dot-pkgs
cd external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests
doctest vet .
doctest test ./scan/root-failure-isolated
doctest test ./scan/missing-root-error
doctest test ./scan/not-a-directory-error
doctest test ./...   # full scan_repo suite

# ai-critic (from repo root)
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-worktree
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-none
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-skipped
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-max-depth
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-archive
doctest test ./tests/remote-agent-machine-backup/restore/show-meta-git-repos
go test ./server/machinebackup/... -count=1
```

All tests must pass. Rebuild binaries via doctest (session cache) or `go run ./script/build`.