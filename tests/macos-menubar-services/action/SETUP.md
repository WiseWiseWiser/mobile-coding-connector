# Scenario

**Feature**: per-service action button gating

```
pid + desiredRunning + enabled -> CanStopService / ShowEnableAction -> booleans
```

## Preconditions

`Op=action` dispatches to action gating helpers in `macosapp/menubar`.

## Steps

1. Leaf supplies `PID`, `DesiredRunning`, and/or `Enabled`.

## Context

REQUIREMENT section A — action gating leaves.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "action"
	return nil
}
```