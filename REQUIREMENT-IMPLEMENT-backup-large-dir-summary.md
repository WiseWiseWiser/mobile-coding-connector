# Implement: Backup Large Dir Summary + Exclusion Revert

## Context

Tests **sealed** — do not modify `./tests/remote-agent-machine-backup`.

Design: `REQUIREMENT-DESIGN-backup-large-dir-summary.md`

RED on: large-dir-summary, included-fetch-skills, extended-exclusions, path-exclusions, dry-run-matches-archive, large-dir-threshold.

## Implement

1. Remove from `builtinExclusionEntries`: `.config/git-fetch-skill/data`, `.config/confluence-fetch-skill/data`, `.knowledge-index` (keep exclusionConfigVer 1.1)
2. `ParseHumanSize(s string) (int64, error)` — shared parser for 40MB, 50M, 1G, etc.
3. CLI `--large-dir-threshold` on `machine backup`; pass to stream/JSON API
4. `BackupStreamRequest` / `BackupRequest`: `LargeDirThresholdBytes int64`
5. `formatBackupDryRunSummary`: sort DotFiles/DirStats by Bytes desc; append `LARGE SIZE`; emit `LARGE DIR DETAIL` using `file/analyse` for large dirs
6. `cmd/agentcli/machine.go`: TTY red for `LARGE SIZE` in verbatim log printer (`golang.org/x/term`)
7. Ensure `BuildPlan` + `WriteArchive` share same walk (dry-run ≡ archive)
8. Update `exclusions_test.go`, `excluded_stats_test.go` as needed

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
git diff ./tests/remote-agent-machine-backup
```