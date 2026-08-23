# Scenario

**Feature**: permanent 502 on chunk 0 exhausts three attempts

```
chunk[0] x3 -> error; complete never called
```

## Preconditions

Inherited from `retry-exhaustion/SETUP.md`.

## Steps

No additional setup.

## Context

Leaf asserts attempt cap and failure.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.AlwaysFailChunk != 0 || req.MaxChunkAttempts != 3 {
		t.Fatalf("parent setup: AlwaysFailChunk=%d MaxChunkAttempts=%d, want 0/3", req.AlwaysFailChunk, req.MaxChunkAttempts)
	}
	return nil
}
```