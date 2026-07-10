# Scenario

**Feature**: remote macOS menu-bar periodic machine backup (paths, schedule, status, recent, retention, menu, Swift contract)

```
# pure helpers (macosapp/menubar)
server URL + home -> ServerNameFromURL / BackupDir / BackupArchiveFilename
BackupTaskStatus + now -> FormatBackupStatusTitle
BackupFileEntry + now -> FormatBackupEntry / Sort / Prune
enabled / lastFinished / nextRunAt -> ShouldRunOnEnable / ShouldRunDue

# remote Swift app
AICriticApp (remote) -> Backup submenu
  Status: … ▸ Enable | Disable
  Backup Now… | Recent | Reveal in Finder…
download -> POST backup/stream -> archive_token -> local .tar.xz
```

## Preconditions

1. `macosapp/menubar` exports backup helpers:
   `ServerNameFromURL`, `BackupDir`, `BackupArchiveFilename`,
   `BackupTaskStatus`, `BackupPhase`, `BackupFileEntry`,
   `FormatBackupStatusTitle`, `FormatBackupEntry`, `FormatBackupRecentEmptyLabel`,
   `ShouldRunOnEnable`, `ShouldRunDue`, `BackupIntervalSeconds`,
   `SortBackupEntriesNewestFirst`, `PruneBackupFiles`,
   `BackupStatusMenuChildren`, `BackupEnableItemEnabled`, `BackupDisableItemEnabled`.
2. Go helper leaves are pure (fixed `now`, entry structs) — no network or live download.
3. Client leaves inspect Swift under `macos-ai-critic/ai-critic-remote-macos/`
   (and optional Shared helpers).
4. Default task state is **off**; interval is **3600s**; archives are **`.tar.xz`**.

## Steps

1. Leaf `Setup` sets `Op` and scenario inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to helpers or source inspection.
3. Leaf `Assert` checks exact strings, booleans, path sets, or Swift contract flags.

## Context

Implements REQUIREMENT-DESIGN-macos-remote-menubar-periodic-backup.md (spec 0.0.2).
Primary logic lives in Go (`macosapp/menubar`); Swift mirrors formatters and wires
download + menu. Local app (`ai-critic-macos`) is **out of scope**.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root: no shared mutation; leaves set Op and inputs.
	// Keep non-stub shape for harness consistency.
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
