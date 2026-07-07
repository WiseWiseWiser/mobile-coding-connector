# Implement: Machine Backup EXCLUDED Section Sizes

## Context

Tests are **sealed** — do not modify `./tests/remote-agent-machine-backup` or
`REQUIREMENT-DESIGN-excluded-sizes.md`.

Design: `REQUIREMENT-DESIGN-excluded-sizes.md`

RED: `backup/excluded-sizes` and `backup/dry-run` fail on missing EXCLUDED stats.

## Feature summary

1. Extend `ExcludePathEntry` with `Files int`, `Bytes int64`
2. During walk, when skipping a regular file, attribute bytes/files to first
   matching rule key (path from `ReasonFor` / rule identifier)
3. After walk, populate stats on each entry in `ExcludedList`; sort by Bytes desc, Path asc
4. Stream phase (`stream.go`): emit column header + one `excluded` progress line per rule
   with Detail containing `FILES SIZE REASON` or structured fields
5. Summary (`stream_summary.go`): `EXCLUDED (N paths, F files, SIZE)` + same table
6. CLI (`cmd/agentcli/machine.go` `printMachineBackupProgress`): print 4-column excluded rows
7. Unit tests in `server/machinebackup/` for attribution, sort, format

## Output format

```
EXCLUDED (24 paths, 847 files, 3.74 GB):
  RULE                                    FILES       SIZE   REASON
  .config/git-fetch-skill/data              412      2.10 GB  git-fetch-skill data cache
  ...
```

Stream: section `EXCLUDED`, then header row, then one line per rule (size-desc).

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
git diff ./tests/remote-agent-machine-backup   # must be clean
```