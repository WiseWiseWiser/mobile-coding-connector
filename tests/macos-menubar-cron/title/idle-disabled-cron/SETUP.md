# Scenario

**Feature**: idle disabled + cron expression title

```
FormatCronTaskTitle("nightly","idle",false,"cron","","0 1 * * *") -> "nightly ○ Idle (disabled) · cron 0 1 * * *"
```

## Preconditions

Task fields match the title contract for this status and schedule mode.

## Steps

1. Set title inputs for this leaf.

## Context

REQUIREMENT leaf: `title/idle-disabled-cron`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "nightly"
	req.Status = "idle"
	req.Enabled = false
	req.ScheduleMode = "cron"
	req.Interval = ""
	req.CronExpr = "0 1 * * *"
	return nil
}
```
