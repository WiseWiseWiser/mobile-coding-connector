# Scenario

**Feature**: Serve + Adhoc lifecycle — command then cancel clears session and port

```
# lifecycle
Start(ctx) -> Alive + command ok
cancel ctx -> session cleared or !Alive; relay port closed
```

## Preconditions

- Scenario: `through-relay-serve-cancel`.
- Same compose as remote-command leaf plus teardown asserts.

## Steps

1. Run full stack command (echo hello).
2. After cancel: Load session nil/!Alive; dial former LocalPort fails.

## Context

- Aligns with P2 serve lifecycle; adds real SSH command before cancel.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioRelayServeCancel
	req.RemoteCommand = "echo hello"
	req.RemoteArgv = []string{"echo", "hello"}
	req.EchoNeedle = "hello"
	return nil
}
```
