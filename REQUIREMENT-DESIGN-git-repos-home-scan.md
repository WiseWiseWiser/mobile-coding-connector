# GIT REPOS: Scan from HOME, List All Repos

## Summary

Change `ScanGitRepos` to discover git repositories with a **single scan from
server `HOME` (`~`)**, listing **every** repo found in `GIT REPOS` (dry-run
summary + archive JSON). Backup `--exclude` / exclusion rules **do not**
affect git scan.

User approved: `/doctest-tdd go ahead` with `--exclude does not affect git scan`.

## Locked decisions

| Item | Decision |
|------|----------|
| Scan root | Single root: absolute `HOME` |
| GIT REPOS contents | All repos `scan_repo` discovers under `HOME` (dot-dir or not) |
| `--exclude` / `ExclusionRules` | **No effect** on git scan (`IgnoreDirs` from exclusions removed) |
| `--skip-git-dirs-scan` | Unchanged — `(skipped)`, no JSON |
| `--git-dirs-scan-max-depth` | Unchanged flag; depth measured from **`HOME`** |
| `scan_repo` defaults | Keep default basename ignores (`node_modules`, `.git`, etc.) |
| JSON / summary format | Unchanged (`version` 1.0, paths relative to HOME, worktree nesting) |
| Error durability | Unchanged — per-entry `error`, scan never aborts backup |

## Data model

Unchanged `GitRepoWorktreesSnapshot` / `git-repo-worktrees.json`.

Path examples after change:

```
  GIT REPOS:
    .wrk-test/main
      branch main  abc1234  clean
      backup git fixture
    projects/demo
      branch main  def5678  clean
      non-dot fixture
```

## Implementation guidance

`server/machinebackup/git_repos.go`:

- Replace `gitScanRoots(home, dirStats)` with `[]string{home}`.
- Remove `gitIgnoreDirsForRoot` usage (or pass empty `IgnoreDirs`).
- `ScanGitRepos` signature may drop `dirStats` and `rules` if unused; update callers in `backup.go`.

CLI stdout ends with `\n` after last content line.

## Test strategy

### New doctest leaves

| Leaf | Scenario | Expected |
|------|----------|----------|
| `backup/git-repos-non-dot-path` | Git repo at `~/projects/demo` (non-dot top-level path) | `GIT REPOS` lists `projects/demo` |
| `backup/git-repos-exclude-independent` | `SeedGitRepos` + `--exclude .wrk-test/**` | `GIT REPOS` still lists `.wrk-test/main`; backup may exclude `.wrk-test` from archive |

### Harness additions (`tests/remote-agent-machine-backup/`)

```go
SeedGitReposNonDot bool  // seeds ~/projects/demo git repo
```

Update root `DOCTEST.md` index + `Run` seeding branch.

### Regression (must stay GREEN)

All existing `backup/git-repos-*` leaves and `restore/show-meta-git-repos`.

Note: `git-repos-max-depth` depth is now from `HOME`; fixture at
`.wrk-test/a/b/c/deep-repo` with `--git-dirs-scan-max-depth 2` should still yield
`(none)` — verify leaf still passes; amend ASSERT only if depth semantics differ.

## Verification

```sh
doctest vet ./tests/remote-agent-machine-backup/backup/git-repos-non-dot-path
doctest vet ./tests/remote-agent-machine-backup/backup/git-repos-exclude-independent
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-non-dot-path
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-exclude-independent
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-...
go test ./server/machinebackup/... -count=1
```

## Approved

User `/doctest-tdd go ahead` with exclude-independent git scan.