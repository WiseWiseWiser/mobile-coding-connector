# Scenario

**Feature**: error event terminates stream with message

```
# type=error ends stream immediately
{"type":"error","message":"upstream_proxy is not configured"} -> err
```

## Preconditions

Mock emits single error frame.

## Steps

1. Set `MockEvents` to one `error` event.

## Context

Matches requirement error envelope semantics.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = []map[string]any{
		{"type": "error", "message": "upstream_proxy is not configured"},
	}
	return nil
}
```
