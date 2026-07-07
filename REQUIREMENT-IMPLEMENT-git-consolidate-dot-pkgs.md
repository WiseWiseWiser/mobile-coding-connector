# Implement: Git Consolidation into dot-pkgs (PR-1–4)

## Context

Consolidate git enrichment from ai-critic `machinebackup/git_repos.go` into
dot-pkgs packages. Design: `REQUIREMENT-DESIGN-git-consolidate-dot-pkgs.md`.

## Tests sealed — do not modify

Dot-pkgs doctest trees (under `external/`, gitignored — treat as sealed):

- `external/dot-pkgs-master-2026-07-07/go-pkgs/git/cmd/tests/`
- `external/dot-pkgs-master-2026-07-07/go-pkgs/git/status/tests/`
- `external/dot-pkgs-master-2026-07-07/go-pkgs/git/checkout/tests/`
- `external/dot-pkgs-master-2026-07-07/go-pkgs/git/reposnapshot/tests/`

ai-critic `tests/remote-agent-machine-backup/` — do not modify.

## Implementation order

### 1. `git/cmd` package
- `cmd.go` with Run, RunOptional, RunCombined
- GIT_OPTIONAL_LOCKS=0, error normalization

### 2. `git/status` package
- Counts, ParsePorcelain, Format with FormatBackup style
- Match backup-dirty doctest exactly

### 3. `git/checkout` package
- Meta, Enrich (durable, partial, unborn HEAD)
- Uses git/cmd + git/status

### 4. `git/reposnapshot` package
- Node, Snapshot, Build from scan_repo.Result
- Main+worktree nesting, RootErrors synthetic nodes

### 5. Refactor `scan_repo`
- Use git/cmd instead of private gitOutput

### 6. Slim `server/machinebackup/git_repos.go`
- ScanGitRepos → scan_repo.Scan → reposnapshot.Build → map to GitRepoWorktreesSnapshot
- Delete enrichGitCheckout, gitOutput, porcelain code, collectWorktreePaths, most buildGitReposSnapshot
- Keep formatGitReposSummaryLines, types, ignore wiring
- Zero exec.Command git in machinebackup

### 7. Update `git_repos_test.go`
- Remove moved tests; keep integration test or point to dot-pkgs

## Verify

```sh
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/cmd/tests/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/status/tests/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/checkout/tests/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/reposnapshot/tests/...
doctest test ./external/dot-pkgs-master-2026-07-07/go-pkgs/git/scan_repo/tests/...

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

All must pass. wrkcli out of scope.