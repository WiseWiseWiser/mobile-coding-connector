# Scenario

**Feature**: ShouldShowBackupProgressWindow — open UI only for interactive runs

```
triggeredBySchedule=false -> true   # Backup Now, enable-immediate
triggeredBySchedule=true  -> false  # hourly tick silent
```

## Preconditions

`Op=show_window`.

## Steps

1. Leaf sets TriggeredBySchedule.

## Context

REQUIREMENT scenarios 6–7; goals 3–4.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "show_window"
	return nil
}
```
