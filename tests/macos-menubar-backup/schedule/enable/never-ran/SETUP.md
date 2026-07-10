# Scenario

**Feature**: enable when never ran allows immediate backup

```
ShouldRunOnEnable(zero, now, 1h) -> true
```

## Preconditions

No successful finish recorded for this server.

## Steps

1. Leave `LastFinishedRFC3339` empty; interval 3600.

## Context

REQUIREMENT #2.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.LastFinishedRFC3339 = ""
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.IntervalSeconds = 3600
	return nil
}
```
