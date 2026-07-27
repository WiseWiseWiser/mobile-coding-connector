# Scenario

**Feature**: Collect discovers PID via lsof on extra port

```
fake opencode serve --port N -> Collect(extraPorts=N) -> fake child PID
```

## Preconditions

- Fake opencode subprocess listening on ephemeral port.

## Steps

1. `StartFakeOpenCode = true`.

## Context

Port discovery path used when registry is stale or for harness stopServer.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.StartFakeOpenCode = true
	return nil
}
```
