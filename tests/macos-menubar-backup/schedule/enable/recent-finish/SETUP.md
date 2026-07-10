# Scenario

**Feature**: enable when last finish was 30m ago does not run now

```
ShouldRunOnEnable(now-30m, now, 1h) -> false
```

## Preconditions

Last successful finish within the interval window.

## Steps

1. Set last finished to 30 minutes before fixed now.

## Context

REQUIREMENT #3.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// now = 15:00Z; last = 14:30Z (30m ago)
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.LastFinishedRFC3339 = "2026-07-10T14:30:00Z"
	req.IntervalSeconds = 3600
	return nil
}
```
