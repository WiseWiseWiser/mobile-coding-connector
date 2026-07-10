# Scenario

**Feature**: disabled tasks never schedule; enable restores firing

```
# seed disabled interval 1s
pre-wait: no runs -> enable -> wait: runs appear
```

## Preconditions

1. Seed task `gated` with `enabled=false`, interval `1s`, short command.
2. PreWait 3s with CapturePreSnapshot (expect 0 runs).
3. Action enable; then poll for ≥1 run.

## Steps

1. Seed disabled short-interval task.
2. Wait pre period; capture pre history count.
3. Enable; poll for runs.

## Context

Priority leaf: enable/disable.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.SeedTasks = []TaskSeed{
		{
			ID:           "gated",
			Name:         "gated",
			Command:      "echo gated-ran",
			ScheduleMode: "interval",
			Interval:     "1s",
			Timeout:      "30s",
			Enabled:      boolPtr(false),
		},
	}
	req.Target = "gated"
	req.PreWaitSecs = 3
	req.CapturePreSnapshot = true
	req.Action = "enable"
	req.PollRunsMin = 1
	req.PollTimeoutSecs = 12
	return nil
}
```
