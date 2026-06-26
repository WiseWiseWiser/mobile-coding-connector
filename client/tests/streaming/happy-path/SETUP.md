# Scenario

**Feature**: Client.Stream happy-path decoding

```
# mock SSE ends with done; Stream returns nil and populated Done map
mock SSE (progress*) -> done -> StreamResult
```

## Preconditions

`MockEvents` sequence ends with `type: done` unless a leaf overrides for order tests.

## Steps

Leaves supply `MockEvents` or rely on root default (3 progress + done).

## Context

Covers requirement scenario `client-stream-consumes-events` and extensions.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if len(req.MockEvents) == 0 {
		req.MockEvents = defaultProgressDoneSequence()
	}
	return nil
}
```
