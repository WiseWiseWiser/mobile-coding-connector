# Scenario

**Feature**: scheduled hourly runs stay silent (no progress window)

```
ShouldShowBackupProgressWindow(triggeredBySchedule=true) -> false
```

## Preconditions

Invocation is the hourly due tick (`checkBackupDue` / schedule path).

## Steps

1. TriggeredBySchedule=true.

## Context

REQUIREMENT #7; goal 4.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.TriggeredBySchedule = true
	return nil
}
```
