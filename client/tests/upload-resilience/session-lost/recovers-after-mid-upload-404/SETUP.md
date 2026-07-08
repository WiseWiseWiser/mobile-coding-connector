# Scenario

**Bug**: 404 mid-upload should recover, not abort at 73%

```
session drop after chunk 28 -> client re-inits -> upload completes
```

## Preconditions

Inherited from `session-lost/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts upload completes despite session loss.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.SessionDropAfterChunk != 28 {
		t.Fatalf("SessionDropAfterChunk=%d, want 28", req.SessionDropAfterChunk)
	}
	return nil
}
```