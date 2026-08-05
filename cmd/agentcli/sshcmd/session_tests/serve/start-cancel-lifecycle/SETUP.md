# Scenario

**Feature**: Start writes Alive session; echo works; cancel clears session and port

```
# full serve lifecycle with injectable Dial
ServeService{Store,Dial,ConfigDir}.Start(ctx)
  -> session file Alive, LocalPort > 0
  -> client echo "hi" through port
cancel ctx
  -> Load nil or !Alive; further dial fails; ssh_config mentions port
```

## Preconditions

- Scenario: `serve-lifecycle`.
- DialFunc is harness echo (not a live remote agent).
- ConfigDir absolute under case dir.

## Steps

1. Start ServeService in a goroutine with cancelable context.
2. Wait until Load returns Alive session with LocalPort > 0.
3. Echo payload through LocalPort.
4. Cancel; wait Start return; assert session cleared and port closed.
5. Assert ssh_config exists and mentions LocalPort.

## Context

- After teardown, P1 client gate would fail (no Alive session).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioServeLifecycle
	req.EchoPayload = "hi"
	return nil
}
```
