# Scenario

**Feature**: idle enabled + cron expression title

```
FormatCronTaskTitle("nightly","idle",true,"cron","","0 1 * * *") -> "nightly ○ Idle · cron 0 1 * * *"
```

## Preconditions

Task fields match the title contract for this status and schedule mode.

## Steps

1. Set title inputs for this leaf.

## Context

REQUIREMENT leaf: `title/idle-cron`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "nightly"
	req.Status = "idle"
	req.Enabled = true
	req.ScheduleMode = "cron"
	req.Interval = ""
	req.CronExpr = "0 1 * * *"
	return nil
}
```
