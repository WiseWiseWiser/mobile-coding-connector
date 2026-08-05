# Scenario

**Feature**: LocalRelay — localhost listen and bidirectional splice via DialFunc

```
# client <-> LocalRelay <-> DialFunc remote
client dial 127.0.0.1:LocalPort -> Accept -> DialFunc() -> io.Copy both ways
Close -> listener and conns stopped; further dial fails
```

## Preconditions

- Scenario family is local-relay.
- DialFunc is injected by harness (echo pipe); no remote agent.

## Steps

1. Leaves set Scenario to echo or close-rejects.
2. Run starts LocalRelay, exercises client dial / close.

## Context

- Parallel-safe: ephemeral port per leaf; no shared globals.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	return nil
}
```
