# Scenario

**Feature**: Backup progress window is a quiet, low-CPU scrollable console

```
# quiet open (no focus steal)
BackupProgressWindow.open
  -> orderFrontRegardless / quiet orderFront
  # MUST NOT NSApp.activate(ignoringOtherApps:)

# batched append (low UI CPU)
onProgress / ProgressSession.append(line)
  -> pendingLines buffer
  -> flush ~100–200ms
  -> textStorage.append(batch) + scrollToEnd once

# keep scrollable console
NSScrollView + NSTextView (isEditable=false, isSelectable=true)

# optional pure helpers (macosapp/menubar)
BackupProgressFlushIntervalMilliseconds
JoinBackupProgressBatch(lines)
ShouldScrollBackupProgressOnFlush()
```

## Preconditions

1. Primary Swift surface: `macos-ai-critic/Shared/BackupProgressWindow.swift`.
2. Hot-path wiring also inspected in
   `macos-ai-critic/ai-critic-remote-macos/AICriticApp.swift` and
   `macos-ai-critic/Shared/MachineBackupClient.swift`.
3. Optional pure helpers live in `macosapp/menubar` (interval / join / scroll policy).
4. Sibling trees `tests/macos-menubar-backup-progress/` and
   `tests/macos-menubar-backup/` stay sealed — not modified by this feature.
5. No UI automation; no network. Client leaves are read-only source contracts.
6. Pre-fix RED expected for quiet-open (activate present) and batch-append
   (per-line `string +=` + scroll).

## Steps

1. Leaf `Setup` sets `Op` (`client` or `helper_*`) and leaf inputs.
2. Root `Run` greps Swift or calls pure Go helpers.
3. Leaf `Assert` checks boolean contract flags or helper outputs.

## Context

Implements REQUIREMENT-DESIGN-macos-menubar-backup-progress-quiet.md (spec 0.0.2).
Focus: open presentation + append CPU policy only. Line format helpers and
show-window schedule policy remain in the sibling progress tree.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
