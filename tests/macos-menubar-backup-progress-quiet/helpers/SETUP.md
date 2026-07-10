# Scenario

**Feature**: optional pure Go helpers document flush interval / batch join / scroll policy

```
# macosapp/menubar
BackupProgressFlushIntervalMilliseconds  # 100..200, canonical 150
JoinBackupProgressBatch(lines)           # "\n" join + trailing newline
ShouldScrollBackupProgressOnFlush()      # true (v1)
```

## Preconditions

Helpers are pure; no network or AppKit. Implementer adds them alongside Swift
batching (optional but sealed once leaves exist).

## Steps

1. Leaf sets `Op=helper_*` and any batch inputs.

## Context

REQUIREMENT optional pure helpers; RED until symbols exist in `macosapp/menubar`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Grouping: ensure request exists; leaves set helper_* Op.
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
