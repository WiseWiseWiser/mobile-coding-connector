# Scenario

**Feature**: periodic backup interval constant

```
BackupIntervalSeconds -> 3600
```

## Preconditions

`Op=schedule_interval`.

## Steps

1. Leaf reads package constant (no extra inputs).

## Context

REQUIREMENT: interval 1 hour.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "schedule_interval"
	return nil
}
```
