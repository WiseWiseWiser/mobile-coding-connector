# REQUIREMENT-DESIGN: set-config merge persisted excludes

## Problem

`--set-config --exclude PATH` replaces entire `backup-config.json`, wiping prior
`exclude_paths` and `large_dir_threshold` when not re-sent. Data loss on incremental updates.

## Fix

### `--set-config --exclude` → merge

1. Load existing `~/.ai-critic/backup-config.json` (if present)
2. Union new `--exclude` paths into `exclude_paths` (add if missing; same path → refresh entry with empty reason)
3. Preserve existing paths not mentioned in this invocation
4. Write merged file back

### `--set-config --large-dir-threshold` → replace only when provided

- Flag provided → update `large_dir_threshold` in file
- Flag omitted → keep existing threshold from file

### Threshold-only set-config

`--set-config --large-dir-threshold 100MB` without `--exclude` must NOT wipe existing `exclude_paths`.

### Exclude-only set-config

`--set-config --exclude .docker` without threshold must NOT wipe existing `large_dir_threshold`.

### Unchanged

- `--include` still blocked on `--set-config` (per-run only)
- Runtime merge builtin + file + CLI unchanged

## Scenarios

| Leaf | Description |
|------|-------------|
| `backup/set-config-merge` | **New**: prereq set `.knowledge-hub`; second set-config adds `.docker`; file has both |
| `backup/set-config-merge-threshold` | **New**: prereq set exclude + threshold; second set-config adds exclude only; threshold preserved |
| `backup/set-config-threshold-only` | **Update if needed**: threshold-only must not wipe prior excludes |

## Verify

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```