# REQUIREMENT-IMPLEMENT: backup-config refinements

## Context

Persisted `~/.ai-critic/backup-config.json` and `--set-config` exist. Followup revision 2 adds validation, `large_dir_threshold` in config file, and `"from user config"` display reasons.

Design: `REQUIREMENT-DESIGN-backup-config-refinements.md`

## Tests are sealed — do not modify

`./tests/remote-agent-machine-backup/` doctest tree is sealed. Implement code only.

## Feature summary

1. **ExclusionConfig** extended with optional `large_dir_threshold` string (human-readable).
2. **SaveUserBackupConfig** persists excludes + threshold; CLI `--set-config` writes empty/omitted reason for excludes.
3. **Display layer** for effective config: empty user reason → `"from user config"`; manual reason preserved.
4. **Threshold resolution**: user config threshold when CLI omits `--large-dir-threshold`; CLI wins per-run.
5. **CLI validation** (`cmd/agentcli/machine.go`):
   - `--set-config` requires `--exclude` and/or `--large-dir-threshold`
   - Mutually exclusive with `--dry-run`, `--show-config`, `--output`, `--include`
6. **API** PUT backup-config accepts `large_dir_threshold`; reject empty body.
7. **Client** `MachineBackupSetConfig` sends threshold.

## Test tree (34 leaves)

New/updated leaves to pass:

- `backup/set-config` — empty persisted reason; effective stdout
- `backup/set-config-threshold` — persist `large_dir_threshold: "100MB"`
- `backup/set-config-empty` — bare `--set-config` errors
- `backup/set-config-mutual-exclude` — `--set-config --dry-run` errors
- `backup/persisted-threshold` — prereq 100MB, dry-run 50MB dir no LARGE SIZE
- `backup/show-config-persisted` — `from user config` + manual reason preserved
- `backup/persisted-merge` — follow-up show-config `from user config`

## Key files

- `server/machinebackup/exclusions.go` — config struct, save/load, merge display reasons
- `server/machinebackup/types.go` — BackupConfigRequest
- `server/machinebackup/api.go` — PUT validation
- `server/machinebackup/stream_summary.go` / backup stream — threshold from user config
- `cmd/agentcli/machine.go` — set-config validation, pass threshold to API
- `client/machine_backup.go` — set config API body

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest vet ./tests/remote-agent-machine-backup
doctest test ./tests/remote-agent-machine-backup/...
```

All 34 doctest leaves must be GREEN.