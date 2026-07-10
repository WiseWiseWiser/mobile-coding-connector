# Scenario

**Feature**: FormatBackupProgress* sealed display lines

```
# SSE + local phases -> human lines in BackupProgressWindow
start / section / progress / log / error / done / download / wrote / Status:*
```

## Preconditions

Format ops produce exact sealed strings (implementer matches ASSERT wants).

## Steps

1. Leaf sets Op and format inputs.

## Context

REQUIREMENT scenarios 8–13 and format table; guard early failures in window.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Grouping only; each leaf sets its own Op.
	if req == nil {
		t.Fatal("req is nil")
	}
	return nil
}
```
