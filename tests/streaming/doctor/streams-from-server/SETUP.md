# Scenario

**Feature**: doctor stdout contains streamed server checks before exit

```
# read stdout pipe line-by-line; checks appear before process waits on slow work
remote-agent stdout -> [ok]/[fail]/[skip] lines -> Result: *
```

## Preconditions

Default integration harness (no upstream fetch delay).

## Steps

1. `UpstreamFetchDelayMs = 0`.

## Context

Requirement scenario 7 — `doctor-integration-streams-from-server`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.UpstreamFetchDelayMs = 0
	return nil
}
```
