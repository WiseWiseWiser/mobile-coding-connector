# Scenario

**Feature**: manual run fires immediately without waiting for schedule

```
# interval 1h (would not fire soon)
POST /api/cron-tasks/run?id= -> history gains a run quickly
```

## Preconditions

1. Seed or create enabled interval task with long interval (`1h`) and short command.
2. Action `run` then poll for ≥1 history entry within a few seconds.

## Steps

1. Seed long-interval task `manual-target`.
2. POST run; poll history.

## Context

Priority leaf: manual run. Still skip if already running (not covered here — task is idle).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTasks = []TaskSeed{
		{
			ID:           "manual-target",
			Name:         "manual-target",
			Command:      "echo manual-fire",
			ScheduleMode: "interval",
			Interval:     "1h",
			Timeout:      "30s",
		},
	}
	req.Target = "manual-target"
	req.Action = "run"
	req.PollRunsMin = 1
	req.PollTimeoutSecs = 10
	return nil
}
```
