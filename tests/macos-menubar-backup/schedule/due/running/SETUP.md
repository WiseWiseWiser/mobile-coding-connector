# Scenario

**Feature**: do not start a second backup while one is running

```
ShouldRunDue(true, true, next<=now, now) -> false
```

## Preconditions

Task enabled and currently in running phase.

## Steps

1. enabled=true, running=true, next in the past.

## Context

REQUIREMENT #6.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	req.Running = true
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.NextRunRFC3339 = "2026-07-10T14:00:00Z"
	return nil
}
```
