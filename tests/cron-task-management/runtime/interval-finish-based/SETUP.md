# Scenario

**Feature**: interval schedule is finish-based and actually fires

```
# interval 1s, short marker command
create enabled interval task -> tick runs command -> finish -> next ≥ finish+1s
# evidence: ≥2 history runs and/or marker lines
```

## Preconditions

1. Interval `1s` and near-instant command (append to marker).
2. Poll until at least 2 completed/started runs within ~15s.

## Steps

1. Create interval task with `UseMarker`.
2. Poll history for ≥2 runs.
3. Assert run count and (when available) `nextRunAt` ≥ last finish + interval.

## Context

Priority leaf: interval finish-based next run + real execution evidence.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "interval-marker"
	req.ScheduleMode = "interval"
	req.Interval = "1s"
	req.Timeout = "30s"
	req.UseMarker = true
	req.Command = "true"
	req.PollRunsMin = 2
	req.PollTimeoutSecs = 15
	return nil
}
```
