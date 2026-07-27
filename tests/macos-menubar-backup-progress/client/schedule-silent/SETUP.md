# Scenario

**Feature**: scheduled due path does not open progress window

```
checkBackupDue -> runBackupNow(triggeredBySchedule: true) -> no window
```

## Preconditions

Hourly tick remains silent by default (ShouldShowBackupProgressWindow false).

## Steps

1. ClientLeaf=schedule-silent.

## Context

REQUIREMENT #20; goal 4.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "schedule-silent"
	return nil
}
```
