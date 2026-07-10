# Scenario

**Feature**: ShouldRunOnEnable — immediate run vs enable-only

```
never ran | finish within 1h | finish older than 1h -> bool
```

## Preconditions

`Op=schedule_on_enable`. Interval 3600s. Fixed now from parent unless overridden.

## Steps

1. Leaf sets `LastFinishedRFC3339` (empty = never).

## Context

REQUIREMENT scenarios 2–4.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "schedule_on_enable"
	return nil
}
```
