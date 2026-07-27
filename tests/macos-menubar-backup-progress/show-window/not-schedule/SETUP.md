# Scenario

**Feature**: manual Backup Now and enable-immediate open the progress window

```
ShouldShowBackupProgressWindow(triggeredBySchedule=false) -> true
```

## Preconditions

Invocation is not the silent hourly path (user Backup Now… or enable-triggered
immediate run).

## Steps

1. TriggeredBySchedule=false.

## Context

REQUIREMENT #6; goals 1 and 3.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.TriggeredBySchedule = false
	return nil
}
```
