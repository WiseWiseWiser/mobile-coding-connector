# Scenario

**Feature**: due when enabled, next run reached, not already running

```
ShouldRunDue(true, false, next<=now, now) -> true
```

## Preconditions

Task enabled; next_run_at in the past or equal to now.

## Steps

1. enabled=true, running=false, next=now.

## Context

REQUIREMENT #5.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	req.Running = false
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.NextRunRFC3339 = "2026-07-10T15:00:00Z"
	return nil
}
```
