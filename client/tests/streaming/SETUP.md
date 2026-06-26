# Scenario

**Feature**: client.Stream consumes SSE progress envelopes

```
# mock server returns canned SSE; client decodes into StreamEvent slice
httptest.Server -> Client.Stream -> StreamResult + callback events
```

## Preconditions

- `client.Stream(method, path, body, onEvent)` exists and supports GET streams.
- Mock server emits `data:` JSON lines compatible with the progress envelope.

## Steps

1. Child `Setup` configures `Request.MockEvents` for the scenario.
2. Root `Run` starts mock server, calls `Client.Stream`, records events.

## Context

Transport-layer unit tests — no `streamcmd` or `remote-agent` involved.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Path == "" {
		req.Path = "/mock/stream"
	}
	return nil
}
```
