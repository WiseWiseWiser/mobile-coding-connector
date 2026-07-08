# Scenario

**Feature**: flaky chunk 2 recovers after two 502s

```
# third POST for chunk 2 succeeds; earlier chunks uploaded once each
chunk[0,1] x1 -> chunk[2] x3 -> chunk[3,4] x1 -> complete
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
	if req.FlakyChunkIndex != 2 || req.TransientFails != 2 {
		t.Fatalf("parent setup: FlakyChunkIndex=%d TransientFails=%d, want 2/2", req.FlakyChunkIndex, req.TransientFails)
	}
	return nil
}
```