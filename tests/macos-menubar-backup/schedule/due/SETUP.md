# Scenario

**Feature**: ShouldRunDue — steady-state due check without overlap

```
enabled + !running + nextRunAt<=now -> true
running | disabled -> false
```

## Preconditions

`Op=schedule_due`.

## Steps

1. Leaf sets Enabled, Running, NextRunRFC3339, NowRFC3339.

## Context

REQUIREMENT scenarios 5–7.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "schedule_due"
	return nil
}
```
