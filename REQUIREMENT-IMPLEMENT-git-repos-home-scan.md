# Implement: GIT REPOS HOME Scan

## Context

Design: `REQUIREMENT-DESIGN-git-repos-home-scan.md`.
User: scan from `HOME`, list all repos; `--exclude` does not affect git scan.

## Tests sealed — do not modify

- `tests/remote-agent-machine-backup/backup/git-repos-non-dot-path/`
- `tests/remote-agent-machine-backup/backup/git-repos-exclude-independent/`
- `tests/remote-agent-machine-backup/DOCTEST.md` (index + Request fields)
- `tests/remote-agent-machine-backup/SETUP.md` (seed helpers)

## Prerequisite: doctest build blocker

All `remote-agent-machine-backup` doctests fail to compile:
`script/lib/commit_msg.go:61` — `gitrunner.Commit(msg)` needs second `bool` arg.
Fix minimally so doctest harness builds (one-line if that's all).

## Implementation

`server/machinebackup/git_repos.go`:

1. `ScanGitRepos` — single root `home` for `scan_repo.Scan`
2. Remove `gitIgnoreDirsForRoot` / pass empty `IgnoreDirs` (exclude-independent)
3. Simplify signature: drop unused `dirStats` and `rules` params if callers allow
4. Update `backup.go` call sites

Keep: `--skip-git-dirs-scan`, max-depth, enrichment, summary format, JSON archive.

## Verify

```sh
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-non-dot-path
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-exclude-independent
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-none
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-max-depth
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-worktree
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-empty-repo
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-skipped
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-archive
doctest test ./tests/remote-agent-machine-backup/restore/show-meta-git-repos
go test ./server/machinebackup/... -count=1
git diff tests/remote-agent-machine-backup/  # must be clean
```

All must be GREEN. Do not modify sealed doctest trees.