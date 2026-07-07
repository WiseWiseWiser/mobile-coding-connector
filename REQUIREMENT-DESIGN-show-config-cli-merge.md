# REQUIREMENT-DESIGN: show-config CLI merge preview

## Problem

`remote-agent machine backup --show-config --exclude .knowledge-index` returns the same JSON as bare `--show-config`. CLI flags are ignored because GET `/api/remote-agent/machine/backup-config` calls `EffectiveExclusionConfigForHome(home)` with no CLI overrides.

## Fix

Use the **same merge path** as backup/restore for `--show-config` preview:

```
effective = Merge(builtin, remote backup-config.json, CLI flags)
```

### Precedence (unchanged)

1. CLI `--exclude`
2. CLI `--include` (removes from exclude set)
3. Remote `backup-config.json` exclude_paths
4. Builtin

Threshold: CLI `--large-dir-threshold` > remote `large_dir_threshold` > 40MB default.

### Display reasons in effective output

| Source | reason |
|--------|--------|
| Builtin | builtin reason |
| Remote, non-empty | user's reason |
| Remote, empty/omitted | `"from user config"` |
| CLI `--exclude` only | `"user excluded"` |

### Scope

| Command | Behavior |
|---------|----------|
| `backup --show-config` | Merge builtin + remote + optional CLI flags |
| `backup --show-config` alone | Builtin + remote only (no CLI) |
| `restore --show-config` (no archive) | Same merge as backup |
| `restore --show-config <archive>` | Archive snapshot only (unchanged; no CLI merge) |

### API

Extend GET `/api/remote-agent/machine/backup-config` with query params:

- `exclude` (repeatable)
- `include` (repeatable)
- `large_dir_threshold` (optional human size string for preview)

Server calls merge with these params; returns effective JSON + trailing newline on CLI.

### CLI

- Forward parsed `--exclude`, `--include`, `--large-dir-threshold` to GET query when `--show-config`.
- Still mutually exclusive with `--set-config`, `--dry-run`, `--output`.

## Scenarios to test

| Leaf | Description |
|------|-------------|
| `backup/show-config-cli-exclude` | **New**: `--show-config --exclude .knowledge-index` includes path with reason `user excluded` |
| `backup/show-config-cli-include` | **New**: prereq remote exclude `.cache`, `--show-config --include .cache` omits `.cache` |
| `restore/show-config-cli-exclude` | **New**: restore without archive, `--show-config --exclude .knowledge-index` merges CLI |
| `backup/show-config` | Unchanged: bare show-config still has builtin `.cache` |

## Verify

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```

CLI stdout ends with trailing newline after JSON.