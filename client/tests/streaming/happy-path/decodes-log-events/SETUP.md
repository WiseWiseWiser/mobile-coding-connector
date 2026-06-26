# Scenario

**Feature**: legacy log events decode for backward compatibility

```
# existing /start/stream log lines map to StreamEvent.Type=log
{"type":"log","message":"..."} -> StreamEvent{Type:log, Message:...}
```

## Preconditions

Mock sequence uses `log` events (not `progress`).

## Steps

1. Set `MockEvents` to log + done sequence.

## Context

Ensures unified `Stream` supports existing streaming commands.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = []map[string]any{
		{"type": "log", "message": "starting"},
		{"type": "log", "message": "ready"},
		{"type": "done", "healthy": true},
	}
	return nil
}
```
