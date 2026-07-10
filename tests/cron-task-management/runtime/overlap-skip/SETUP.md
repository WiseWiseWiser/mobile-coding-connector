# Scenario

**Feature**: when previous run still live at fire time, skip starting another

```
# interval 1s, command sleep 30
first fire starts sleep -> subsequent due times while live -> skip (one PID / one open run)
```

## Preconditions

1. Long-running command `sleep 30`, interval `1s`, timeout `2m`.
2. Wait ~4s after create so multiple interval dues would fire if overlap allowed.

## Steps

1. Create long-running interval task.
2. Wait 4 seconds while first run is active.
3. Assert still a single running instance (one live PID) and history has only one
   in-progress/started run without a second concurrent start.

## Context

Priority leaf: overlap skip only.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "overlap-sleep"
	req.Command = "sleep 30"
	req.ScheduleMode = "interval"
	req.Interval = "1s"
	req.Timeout = "2m"
	req.WaitSecs = 4
	// Also fetch history after wait
	req.PollRunsMin = 1
	req.PollTimeoutSecs = 10
	return nil
}
```
