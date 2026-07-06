# Scenario

**Feature**: restore apply restores .machine.bak snapshots, skips meta injections

```
# seed ~/.backup/config.json, backup, wipe config, restore apply
.backup/config.json.machine.bak -> ~/.backup/config.json (old content)
```

## Preconditions

`SeedBackupMeta=true` before prereq backup; post-backup wipe of `.backup/config.json`.

## Steps

1. `SeedBackupMeta=true`, `IncludePaths=[".backup"]`, `AfterBackupMutate=wipe-backup-config`.
2. Args: `machine restore` (apply, no `--dry-run`).
3. `--include .backup` re-includes the meta dir so `.machine.bak` snapshots can be applied
   (built-in `.backup` exclusion otherwise skips the restore target).

## Context

REQUIREMENT leaf `restore/meta-restore`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedBackupMeta = true
	req.IncludePaths = []string{".backup"}
	req.AfterBackupMutate = "wipe-backup-config"
	req.Args = []string{"machine", "restore"}
	return nil
}
```