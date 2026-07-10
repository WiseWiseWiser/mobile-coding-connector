# Scenario

**Feature**: enable-on-run policy, due checks, and 1h interval

```
lastFinished + now + interval -> ShouldRunOnEnable
enabled + running + nextRunAt + now -> ShouldRunDue
BackupIntervalSeconds -> 3600
```

## Preconditions

Interval default is 1 hour (3600s). No network.

## Steps

1. Leaf sets `Op` to `schedule_on_enable`, `schedule_due`, or `schedule_interval`.

## Context

REQUIREMENT: enable/schedule policy scenarios 1–7 + interval.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.NowRFC3339 == "" {
		req.NowRFC3339 = defaultNowRFC3339
	}
	if req.IntervalSeconds == 0 {
		req.IntervalSeconds = 3600
	}
	return nil
}
```
