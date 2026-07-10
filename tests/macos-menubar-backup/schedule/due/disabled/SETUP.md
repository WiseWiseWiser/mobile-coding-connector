# Scenario

**Feature**: disabled task never runs on due check

```
ShouldRunDue(false, false, next<=now, now) -> false
```

## Preconditions

Task default/off or user disabled.

## Steps

1. enabled=false, running=false, next due.

## Context

REQUIREMENT #7 (and default off steady-state).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = false
	req.Running = false
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.NextRunRFC3339 = "2026-07-10T14:00:00Z"
	return nil
}
```
