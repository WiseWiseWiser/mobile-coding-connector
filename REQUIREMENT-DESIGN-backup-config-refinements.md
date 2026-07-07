# REQUIREMENT-DESIGN: backup-config refinements

## Summary

Extend persisted `~/.ai-critic/backup-config.json` and `--set-config` behavior per followup revision 2.

## Data model

### Persisted file (`~/.ai-critic/backup-config.json`)

User-authored only (no builtin entries):

```json
{
  "version": "1.1",
  "exclude_paths": [
    { "path": ".knowledge-hub" },
    { "path": ".knowledge-index", "reason": "knowledge index cache" }
  ],
  "large_dir_threshold": "100MB"
}
```

| Field | Type | Rule |
|-------|------|------|
| `version` | string | `"1.1"` |
| `exclude_paths` | array | `path` required; `reason` optional (omit or empty when set via CLI) |
| `large_dir_threshold` | string | Human-readable size (`40MB`, `50M`, `1G`); optional; parsed via `ParseHumanSize` |

### Effective merged output (`--show-config`, GET backup-config)

Builtin + user excludes merged; includes resolved `large_dir_threshold` when user persisted it.

Display reasons in effective output:

| Source | `reason` in effective JSON |
|--------|---------------------------|
| Builtin | Builtin reason unchanged |
| User file, non-empty `reason` | User's reason as written |
| User file, empty/omitted `reason` | `"from user config"` |

### Runtime threshold resolution

```
effectiveThreshold =
  CLI --large-dir-threshold (per-run, wins)
  ?? parse(userConfig.large_dir_threshold)
  ?? 40MB default
```

## CLI rules (`machine backup` only)

### `--set-config`

- **Backup only** — restore has no `--set-config` (unchanged).
- **Requires input** — error if neither `--exclude` nor `--large-dir-threshold` provided.
- **Standalone** — error if combined with `--dry-run`, `--show-config`, `--output`, or `--include`.
- **Allowed inputs**: `--exclude PATH` (repeatable), `--large-dir-threshold SIZE`.
- Writes persisted file with empty/omitted `reason` for CLI-set excludes (not `"user excluded"`).
- Stdout: prints effective merged JSON + trailing newline.

### `--show-config`

Unchanged: effective merged config from server (with display reasons + threshold when set).

## API

### PUT `/api/remote-agent/machine/backup-config`

Body:

```json
{
  "exclude": [".knowledge-hub"],
  "large_dir_threshold": "100MB"
}
```

- Reject empty body (no `exclude` entries and no `large_dir_threshold`).
- Return effective merged config JSON.

### GET `/api/remote-agent/machine/backup-config`

Return effective merged config with display reasons and `large_dir_threshold` when persisted.

## Scenarios to test (doctest leaves)

Extend `tests/remote-agent-machine-backup/`:

| Leaf | Description |
|------|-------------|
| `backup/set-config` | **Update**: persisted file has `.knowledge-hub` with empty/omitted reason (not `user excluded`) |
| `backup/set-config-threshold` | **New**: `--set-config --large-dir-threshold 100MB` persists string; file contains `large_dir_threshold` |
| `backup/set-config-empty` | **New**: bare `--set-config` exits non-zero with error message |
| `backup/set-config-mutual-exclude` | **New**: `--set-config --dry-run` (or `--output`) exits non-zero |
| `backup/persisted-merge` | **Update**: EXCLUDED/dry-run uses persisted excludes without CLI flags; reason shows `from user config` in `--show-config` effective output |
| `backup/persisted-threshold` | **New**: after prereq set-config with threshold, dry-run without CLI threshold respects persisted value (e.g. 100MB suppresses LARGE SIZE on 50MB dir) |
| `backup/show-config-persisted` | **New**: after set-config, `--show-config` shows user path with `from user config` and manual reason preserved when edited |

## Unit tests (implementer may add)

- `SaveUserBackupConfig` stores `large_dir_threshold` string
- `MergeExclusions` display reason `from user config` for empty user reasons
- `ExcludePathsFromStrings` omits reason field
- Threshold resolution from user config when CLI bytes = 0

## Verify commands

```sh
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
go test ./server/machinebackup/... -count=1
```

## Invariants (unchanged)

- Dry-run ≡ real backup merge inputs
- Guard: cannot persist exclude for `.ai-critic` or `backup-config.json`
- `--include` stays CLI-only, not in config file
- Archive `.backup/config.json` remains exclusion-only at pack time