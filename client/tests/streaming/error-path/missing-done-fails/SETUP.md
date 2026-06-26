# Scenario

**Feature**: stream without terminal done fails

```
# progress only, connection closes — Stream must error
progress -> (no done)
```

## Preconditions

Mock emits progress without terminal frame.

## Steps

1. Set `MockEvents` to a lone `progress` event.

## Context

Mirrors existing `postSSEJSON` guard: stream must not succeed silently.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = []map[string]any{
		{"type": "progress", "id": "only", "layer": "server", "name": "lonely", "status": "ok"},
	}
	return nil
}
```
