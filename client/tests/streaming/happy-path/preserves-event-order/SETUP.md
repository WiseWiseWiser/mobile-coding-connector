# Scenario

**Feature**: StreamResult.Events preserves wire order

```
# interleaved types: section, progress, meta, done
wire order == result.Events order
```

## Preconditions

Mock sequence mixes event types.

## Steps

1. Configure explicit `MockEvents` with known order.

## Context

Guards against reordering or dropping frames during SSE scan.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = []map[string]any{
		{"type": "section", "message": "Server checks"},
		{"type": "progress", "id": "p1", "layer": "server", "name": "one", "status": "ok"},
		{"type": "meta", "message": "status"},
		{"type": "progress", "id": "p2", "layer": "server", "name": "two", "status": "ok"},
		{"type": "done", "healthy": false},
	}
	return nil
}
```
