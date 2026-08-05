# Scenario

**Feature**: Through LocalRelay, client runs echo; stdout contains needle

```
# full path command
Serve Dial->Adhoc; runner.Run(echo hello) via session.LocalPort
stdout contains hello
```

## Preconditions

- Scenario: `through-relay-command`.
- RemoteArgv `["echo","hello"]`.

## Steps

1. Compose Adhoc + ServeService + CryptoSSHRunner.
2. Assert command stdout needle; ServeErr empty.

## Context

- Cancel serve after command (cleanup); teardown fields asserted in sibling leaf.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioRelayCommand
	req.RemoteCommand = "echo hello"
	req.RemoteArgv = []string{"echo", "hello"}
	req.EchoNeedle = "hello"
	return nil
}
```
