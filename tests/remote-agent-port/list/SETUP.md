# Scenario

**Feature**: port list of remote listening ports

```
GET /api/ports/local (seeded) -> remote-agent port list -> table or JSON
```

## Preconditions

L2 seeds `LocalPorts` and optional persistent `Forwards`.

## Steps

1. Leaf seeds listeners / forwards and CLI args.
2. Run CLI against L2 mux.
3. Assert stdout shape and exit 0.

## Context

Frontend Services/Ports local table parity (PORT/PID/COMMAND).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	return nil
}
```
