# Scenario

**Feature**: flaky GET recovers after two 502s

```
# third GET succeeds; file intact
GET x3 -> success
```

## Preconditions

Inherited from `transient-recovery/SETUP.md`.

## Steps

No additional setup beyond parent.

## Context

Leaf asserts retry count and successful assembly.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.TransientFails != 2 {
		t.Fatalf("parent setup: TransientFails=%d, want 2", req.TransientFails)
	}
	return nil
}
```