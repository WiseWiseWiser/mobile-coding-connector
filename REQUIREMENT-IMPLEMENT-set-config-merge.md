# REQUIREMENT-IMPLEMENT: set-config merge persisted excludes

## Context

`SaveUserBackupConfig` replaces entire file. `--set-config --exclude` should merge.

Design: `REQUIREMENT-DESIGN-set-config-merge.md`

## Tests sealed — do not modify

`./tests/remote-agent-machine-backup/`

## Implementation

Change `SaveUserBackupConfig` or PUT handler to:

1. Load existing user config
2. Merge new exclude paths into existing `exclude_paths` (union by path)
3. Update `large_dir_threshold` only when non-empty in request; preserve existing when omitted
4. Threshold-only PUT must not clear exclude_paths
5. Exclude-only PUT must not clear large_dir_threshold

Suggested: `MergeUserBackupConfig(existing, newExcludes, newThreshold string) ExclusionConfig`

## RED failures

- `set-config-merge`: second exclude wipes first
- `set-config-merge-threshold`: threshold lost on exclude-only update
- `set-config-threshold`: prereq exclude wiped on threshold-only update

## Verify

```sh
go test ./server/machinebackup/... -count=1
doctest test ./tests/remote-agent-machine-backup/...
```

All leaves GREEN (39+).