# Scenario

**Feature**: Stream delivers progress events then done payload

```
# default fixture: 3 progress + done
Client.Stream -> events[0..2] progress, Done map populated
```

## Preconditions

Root default `MockEvents` (3 progress + done).

## Steps

No additional configuration.

## Context

Requirement scenario 4 — `client-stream-consumes-events`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = defaultProgressDoneSequence()
	return nil
}
```
