# Scenario

**Feature**: Backup Now progress window + one-shot when disabled (remote menubar)

```
# pure helpers (macosapp/menubar)
hasEndpoint + !running + serverName -> CanRunBackupNow  # ignores enabled
triggeredBySchedule -> ShouldShowBackupProgressWindow
SSE frames + local phases -> FormatBackupProgress*

# remote Swift app
AICriticApp (remote) -> Backup Now… / enable-immediate
  -> BackupProgressWindow (append lines) when !schedule
MachineBackupClient -> stream events (not token-only) -> download
hourly tick (triggeredBySchedule=true) -> silent (no window)
```

## Preconditions

1. `macosapp/menubar` exports (or will export) progress helpers:
   `CanRunBackupNow`, `ShouldShowBackupProgressWindow`,
   `FormatBackupProgressStartHeader`, `FormatBackupProgressStartedAt`,
   `FormatBackupProgressWindowTitle`, `FormatBackupProgressSection`,
   `FormatBackupProgressFrame`, `FormatBackupProgressLog`,
   `FormatBackupProgressError`, `FormatBackupProgressDone`,
   `FormatBackupProgressDownloadStart`, `FormatBackupProgressWrote`,
   `FormatBackupProgressStatusSuccess`, `FormatBackupProgressStatusFailed`,
   `FormatBackupProgressGuardError`.
2. Go helper leaves are pure — fixed strings/times; no network or live download.
3. Client leaves inspect Swift under `macos-ai-critic/ai-critic-remote-macos/`
   and `macos-ai-critic/Shared/`.
4. Sealed line styles are exact (including `ERROR:` prefix, `Status: Success` /
   `Status: Failed`, `[section]` / `[progress]` / `[done]`, verbatim log).
5. Sibling tree `tests/macos-menubar-backup/` stays sealed and is not modified.

## Steps

1. Leaf `Setup` sets `Op` and scenario inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to helpers or source inspection.
3. Leaf `Assert` checks exact strings, booleans, or Swift contract flags.

## Context

Implements REQUIREMENT-DESIGN-macos-menubar-backup-progress-window.md (spec 0.0.2).
Primary logic in Go (`macosapp/menubar`); Swift mirrors formatters and wires
progress window + stream callbacks. Local app (`ai-critic-macos`) is **out of
scope**. Hourly schedule remains silent by default.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root: no shared mutation; leaves set Op and inputs.
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
