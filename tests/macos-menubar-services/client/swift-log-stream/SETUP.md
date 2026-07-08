# Scenario

**Feature**: View Logs streams service logs via server SSE

```
Button("View Logs") -> ServerClient -> GET /api/logs/stream?path=...&lines=1000 -> LogTailWindow (SSE)
```

## Preconditions

Floating log window consumes server SSE on `/api/logs/stream` with `lines=1000`;
no local `/usr/bin/tail` or `Process` spawn for service logs.

## Steps

1. Set `ClientLeaf=swift-log-stream`.

## Context

REQUIREMENT leaf: `client/swift-log-stream` (replaces `swift-log-tail`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "swift-log-stream"
	return nil
}
```