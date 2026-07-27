# Scenario

**Feature**: port visit CLI foreground, detach JSON, invalid ports

```
remote-agent port visit <port> [--detach] [--json] [--idle] -> URL/JSON or Error
```

## Preconditions

L2 mux with VisitSessionManager.RegisterAPI; both providers available.

## Steps

1. Leaf sets Args and provider seeds.
2. Op=cli.
3. Assert exit, stdout JSON/URL, residual sessions.

## Context

Foreground default; --detach exits while server keeps session.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	return nil
}
```
