# Scenario

**Feature**: per-cron-task submenu title strings

```
name + status + enabled + scheduleMode/interval/cronExpr -> FormatCronTaskTitle -> title line
```

## Preconditions

`Op=title` dispatches to `menubar.FormatCronTaskTitle`.

## Steps

1. Leaf supplies `Name`, `Status`, `Enabled`, `ScheduleMode`, and schedule fields.

## Context

REQUIREMENT title format (name + glyph + short schedule).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "title"
	return nil
}
```
