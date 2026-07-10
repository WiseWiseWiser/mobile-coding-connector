# Scenario

**Feature**: history API returns recent runs after executions

```
# short interval + marker command produces runs
GET /api/cron-tasks/history?id= -> non-empty array of CronTaskRun (UTC timestamps)
```

## Preconditions

1. Create interval `1s` with quick command.
2. Poll until ≥2 runs then fetch history (Run always attaches History when target known).

## Steps

1. Create enabled short-interval task.
2. PollRunsMin=2.
3. Assert history entries have StartedAt (RFC3339-ish) and length ≥2.

## Context

Priority leaf: history returns recent runs.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.TaskName = "hist-runs"
	req.Command = "echo hist"
	req.ScheduleMode = "interval"
	req.Interval = "1s"
	req.Timeout = "30s"
	req.PollRunsMin = 2
	req.PollTimeoutSecs = 15
	return nil
}
```
