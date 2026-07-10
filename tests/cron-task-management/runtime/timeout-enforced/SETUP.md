# Scenario

**Feature**: timeout always enforced — kill process group past deadline

```
# timeout 2s, command sleep 60
run starts -> timeout fires -> process dead; history/status has timeout error
```

## Preconditions

1. Timeout `2s` shorter than command `sleep 60`.
2. Wait long enough after first fire (~5s) for kill + history write.

## Steps

1. Create interval task with short timeout (interval large so one fire is enough),
   or use manual-friendly short interval.
2. Wait 6s for timeout kill.
3. Assert process dead and error mentions timeout.

## Context

Priority leaf: timeout enforce.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "timeout-sleep"
	req.Command = "sleep 60"
	req.ScheduleMode = "interval"
	req.Interval = "1s"
	req.Timeout = "2s"
	req.WaitSecs = 6
	req.PollRunsMin = 1
	req.PollTimeoutSecs = 12
	return nil
}
```
