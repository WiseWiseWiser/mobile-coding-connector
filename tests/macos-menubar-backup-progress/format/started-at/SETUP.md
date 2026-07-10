# Scenario

**Feature**: start time line uses sealed wall-clock layout

```
FormatBackupProgressStartedAt(2026-07-10T15:00:00Z) -> "Started 2026-07-10 15:00:00"
```

## Preconditions

Format uses the time value’s wall clock as `2006-01-02 15:04:05` (no TZ suffix).

## Steps

1. Op=format_started_at; fixed RFC3339.

## Context

REQUIREMENT format table “Started {local time}”.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "format_started_at"
	req.TimeRFC3339 = "2026-07-10T15:00:00Z"
	return nil
}
```
