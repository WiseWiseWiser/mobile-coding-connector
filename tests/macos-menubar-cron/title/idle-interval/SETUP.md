# Scenario

**Feature**: idle enabled + interval schedule title

```
FormatCronTaskTitle("backup","idle",true,"interval","5m","") -> "backup ○ Idle · every 5m"
```

## Preconditions

Task fields match the title contract for this status and schedule mode.

## Steps

1. Set title inputs for this leaf.

## Context

REQUIREMENT leaf: `title/idle-interval`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "backup"
	req.Status = "idle"
	req.Enabled = true
	req.ScheduleMode = "interval"
	req.Interval = "5m"
	req.CronExpr = ""
	return nil
}
```
