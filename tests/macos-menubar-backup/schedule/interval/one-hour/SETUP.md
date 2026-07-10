# Scenario

**Feature**: sealed 1-hour backup interval

```
menubar.BackupIntervalSeconds == 3600
```

## Preconditions

Steady-state period between automatic runs is one hour.

## Steps

1. Invoke interval op (`schedule_interval`).

## Context

REQUIREMENT runtime policy: Interval 1 hour.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "schedule_interval"
	return nil
}
```
