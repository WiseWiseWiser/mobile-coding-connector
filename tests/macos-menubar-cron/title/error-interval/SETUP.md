# Scenario

**Feature**: error + interval schedule title

```
FormatCronTaskTitle("scrape","error",true,"interval","1m","") -> "scrape ⚠ Error · every 1m"
```

## Preconditions

Task fields match the title contract for this status and schedule mode.

## Steps

1. Set title inputs for this leaf.

## Context

REQUIREMENT leaf: `title/error-interval`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "scrape"
	req.Status = "error"
	req.Enabled = true
	req.ScheduleMode = "interval"
	req.Interval = "1m"
	req.CronExpr = ""
	return nil
}
```
