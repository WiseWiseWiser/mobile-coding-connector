# Scenario

**Feature**: enable/disable NSAlert message (server or fallback)

```
serverMessage -> CronToggleAlertMessage -> alert body; empty -> "Task updated"
```

## Preconditions

`Op=alert` dispatches to `menubar.CronToggleAlertMessage`.

## Steps

1. Leaf supplies `ServerMessage`.

## Context

REQUIREMENT: show alert with server message (or fallback), then refresh.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "alert"
	return nil
}
```
