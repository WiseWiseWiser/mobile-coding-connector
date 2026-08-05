# Scenario

**Feature**: ServeService.Start(ctx) — Alive session, relay, cancel teardown

```
# blocking serve lifecycle
ServeService.Start(ctx) -> LocalRelay + FileSessionStore.Save(Alive)
client echo through LocalPort
cancel ctx -> Clear session + Close relay
```

## Preconditions

- Scenario family is serve.
- Store Root and ConfigDir absolute under d.DOCTEST_CASE.
- Dial injected for lifecycle; intentionally nil for missing-dial.

## Steps

1. Leaves set Scenario to start-cancel-lifecycle or missing-dial.
2. Run exercises Start(ctx) with cancel or misconfiguration.

## Context

- Bridges P1 ServeStarter later; this tree calls Start(ctx) directly (L2).

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
