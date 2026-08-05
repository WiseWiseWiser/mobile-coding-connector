# Scenario

**Feature**: CryptoSSHRunner echo p4-ok through agent tunnel Dial

```
# end-to-end L2
CreateSession -> Serve Dial=SSHTunnelDialFunc -> runner echo p4-ok -> stdout
```

## Preconditions

- Scenario: `through-serve-remote-command`.
- RemoteArgv `["echo","p4-ok"]`; EchoNeedle `p4-ok`.

## Steps

1. Compose client tunnel + ServeService + CryptoSSHRunner.
2. Assert RunnerErr empty; Stdout contains needle; ServeErr empty.

## Context

- Replaces P3 direct `DialTCP(Adhoc)` with client WS Dial (production gap).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioThroughServeCommand
	req.RemoteArgv = []string{"echo", "p4-ok"}
	req.EchoNeedle = "p4-ok"
	return nil
}
```
