# Scenario

**Bug**: second upload should skip 39 cached chunks

```
39/40 chunks on server -> expect 1 chunk POST
```

## Preconditions

Inherited from `cross-run-resume/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts minimal network transfer on resume.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.PrefilledChunks != 39 {
		t.Fatalf("PrefilledChunks=%d, want 39", req.PrefilledChunks)
	}
	return nil
}
```