# Implement: Machine Backup Git Repo / Worktree Meta

## Context

Machine backup dry-run and real backup should discover git repos under included
top-level dot-dirs, capture metadata, show a `GIT REPOS` summary section, and
inject `.backup/git-repo-worktrees.json` into archives.

Design locked in followup + `REQUIREMENT-DESIGN-machine-backup-git-repos.md`.

## Tests are sealed — do not modify

The doctest tree under `tests/remote-agent-machine-backup/` is sealed. Do not
edit any file under that directory unless the orchestrator explicitly approves
a test correction.

## Feature summary

1. **Scan** included dot-dir roots with `scan_repo` (`ListWorktrees: true`,
   `MaxDepth` from `--git-dirs-scan-max-depth`, 0 = unlimited).
2. **Enrich** each repo/worktree: branch, 7-char sha, commit message, porcelain status counts.
3. **Dry-run summary** — emit `GIT REPOS` section (verbatim log frames):
   - repos tree, `(none)`, or `(skipped)` per flags.
4. **Real backup** — write `.backup/git-repo-worktrees.json` at pack time
   (omit when `--skip-git-dirs-scan`).
5. **CLI flags** on `machine backup`:
   - `--skip-git-dirs-scan`
   - `--git-dirs-scan-max-depth N`
6. **Restore** — `isBackupMetaSnapshot` includes new file; `ReadArchiveMeta` picks it up
   for `--show-meta` automatically.

## Test tree (7 new leaves)

| Leaf | Asserts |
|------|---------|
| `backup/git-repos-summary` | Dry-run `GIT REPOS` with main repo |
| `backup/git-repos-worktree` | Worktree nested block + dirty count |
| `backup/git-repos-none` | `GIT REPOS: (none)` |
| `backup/git-repos-skipped` | `(skipped)` + no archive JSON |
| `backup/git-repos-max-depth` | Deep repo excluded at max depth 2 |
| `backup/git-repos-archive` | Archive JSON v1.0 valid |
| `restore/show-meta-git-repos` | `--show-meta` prints git section |

Fixtures use `.wrk-test/` dot-dir under seeded server HOME.

## Implementation hints

### Files likely touched

- `server/machinebackup/types.go` — request fields, snapshot types, plan field
- `server/machinebackup/git_repos.go` (new) — scan, enrich, format summary, build JSON
- `server/machinebackup/stream_summary.go` — call GIT REPOS formatter before TOTAL
- `server/machinebackup/meta.go` — inject `git-repo-worktrees.json`; extend `isBackupMetaSnapshot`
- `server/machinebackup/stream.go` / `backup.go` — pass scan options through plan build
- `server/machinebackup/api.go` — parse new JSON fields
- `cmd/agentcli/machine.go` — CLI flags → request body
- `client/machine_backup.go` — archive request fields if needed

### Status string

`clean` or `dirty (N modified, M added, …)` from `git status --porcelain`.

### Worktree nesting

Main `scan_repo` repos (`RepoTypeMain`) → top-level `repos[]`.
Linked worktrees → nested under parent; separate worktree checkout rows folded in.

### API wiring

```go
SkipGitDirsScan     bool `json:"skip_git_dirs_scan,omitempty"`
GitDirsScanMaxDepth int  `json:"git_dirs_scan_max_depth,omitempty"`
```

Dry-run: `BackupStreamRequest`. Real backup: `BackupRequest` (or shared body).

## Verify

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-summary
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-worktree
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-none
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-skipped
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-max-depth
doctest test ./tests/remote-agent-machine-backup/backup/git-repos-archive
doctest test ./tests/remote-agent-machine-backup/restore/show-meta-git-repos
go test ./server/machinebackup/... -count=1
```

All 7 new leaves must pass; no regressions on existing machine-backup tests.