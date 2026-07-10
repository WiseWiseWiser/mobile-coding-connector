# Scenario

**Feature**: enable-triggered immediate run opens the same progress window

```
setBackupEnabled(true) + shouldRunNow -> runBackupNow(show window)
# must NOT force triggeredBySchedule:true for the interactive enable path
```

## Preconditions

Enable-on-run (never ran / last > 1h) uses the progress window like Backup Now.

## Steps

1. ClientLeaf=enable-immediate-shows-window.

## Context

REQUIREMENT goal 3; scenario enable-immediate (Swift).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "enable-immediate-shows-window"
	return nil
}
```
