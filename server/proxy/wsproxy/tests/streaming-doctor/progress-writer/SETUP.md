# Scenario

**Feature**: progress.Writer emits typed SSE frames

```
# unit test drives Writer directly (no doctor logic)
progress.Writer.EmitProgress/Section/Done -> httptest.ResponseRecorder
```

## Preconditions

`Request.Target` is `progress-writer`.

## Steps

1. Set `req.Target = TargetProgressWriter`.

## Context

Validates the shared streaming framework independent of ws-proxy doctor checks.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Target = TargetProgressWriter
	return nil
}
```
