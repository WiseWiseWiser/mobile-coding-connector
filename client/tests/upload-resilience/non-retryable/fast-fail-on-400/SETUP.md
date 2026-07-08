# Scenario

**Feature**: HTTP 400 on chunk 1 aborts without retry

```
chunk[0] x1 -> chunk[1] x1 (400) -> error
```

## Preconditions

Inherited from `non-retryable/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts single POST on failed chunk.

```go
import (
	"net/http"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.FlakyChunkIndex != 1 || req.PermanentStatus != http.StatusBadRequest {
		t.Fatalf("parent setup: FlakyChunkIndex=%d PermanentStatus=%d, want 1/400", req.FlakyChunkIndex, req.PermanentStatus)
	}
	return nil
}
```