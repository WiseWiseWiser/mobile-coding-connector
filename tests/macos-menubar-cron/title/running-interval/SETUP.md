# Scenario

**Feature**: running + interval schedule title

```
FormatCronTaskTitle("backup","running",true,"interval","5m","") -> "backup ● Running · every 5m"
```

## Preconditions

Task fields match the title contract for this status and schedule mode.

## Steps

1. Set title inputs for this leaf.

## Context

REQUIREMENT leaf: `title/running-interval`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "backup"
	req.Status = "running"
	req.Enabled = true
	req.ScheduleMode = "interval"
	req.Interval = "5m"
	req.CronExpr = ""
	return nil
}
```
