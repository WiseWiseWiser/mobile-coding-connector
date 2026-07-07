# REQUIREMENT-IMPLEMENT: show-config CLI merge preview

## Context

`--show-config` ignores CLI flags. User expects preview of effective config with builtin + remote + CLI merged.

Design: `REQUIREMENT-DESIGN-show-config-cli-merge.md`

## Tests are sealed — do not modify

`./tests/remote-agent-machine-backup/` doctest tree is sealed.

## Implementation

1. **API** `GET /api/remote-agent/machine/backup-config`:
   - Parse query `exclude`, `include`, `large_dir_threshold`
   - Call `ResolveExclusionRules(home, exclude, include)` instead of nil,nil
   - Include resolved threshold in effective output when applicable

2. **Server** Add `EffectiveExclusionConfig(home, exclude, include, largeDirThreshold string)` or extend existing helper

3. **Client** `MachineBackupEffectiveConfig(exclude, include, largeDirThreshold string)` — pass as query params

4. **CLI** `printEffectiveExclusionConfig` — pass parsed flags from `runMachineBackup` and `runMachineRestore` (no archive only)

5. **Restore** `printRestoreConfig` when archive empty — forward exclude/include to effective config GET

## RED failures

- `show-config-cli-exclude`: missing `.knowledge-index` with `user excluded`
- `show-config-cli-include`: `.cache` still in exclude_paths after `--include .cache`
- `restore/show-config-cli-exclude`: same as backup exclude case

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest test ./tests/remote-agent-machine-backup/...
```

All 37 leaves GREEN.