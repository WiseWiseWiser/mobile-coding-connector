# Scenario

**Feature**: Backup Now… opens progress window (not schedule-only)

```
Button("Backup Now…") -> runBackupNow(schedule=false) -> show progress window
```

## Preconditions

Manual path uses showWindow / !triggeredBySchedule.

## Steps

1. ClientLeaf=manual-shows-window.

## Context

REQUIREMENT #17.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "manual-shows-window"
	return nil
}
```
