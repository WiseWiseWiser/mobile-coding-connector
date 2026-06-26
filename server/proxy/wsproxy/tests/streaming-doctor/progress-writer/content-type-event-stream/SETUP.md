# Scenario

**Feature**: progress.Writer sets SSE response headers

```
# first WriteHeader side effect is text/event-stream
progress.NewWriter(rec) -> Content-Type header
```

## Preconditions

Inherited `TargetProgressWriter` from ancestor.

## Steps

No additional setup.

## Context

Ensures consumers can distinguish SSE streams from JSON responses.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Target = TargetProgressWriter
	return nil
}
```
