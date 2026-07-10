# Scenario

**Feature**: enable when last finish was 2h ago runs immediately

```
ShouldRunOnEnable(now-2h, now, 1h) -> true
```

## Preconditions

Last finish older than the 1h interval.

## Steps

1. Set last finished to 2 hours before fixed now.

## Context

REQUIREMENT #4.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// now = 15:00Z; last = 13:00Z (2h ago)
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.LastFinishedRFC3339 = "2026-07-10T13:00:00Z"
	req.IntervalSeconds = 3600
	return nil
}
```
