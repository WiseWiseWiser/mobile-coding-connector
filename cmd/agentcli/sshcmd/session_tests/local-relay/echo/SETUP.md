# Scenario

**Feature**: Relay echoes client payload through DialFunc remote side

```
# byte integrity
LocalRelay.Start -> LocalPort > 0
client writes "hi" -> DialFunc echo side -> client reads "hi"
```

## Preconditions

- Scenario: `relay-echo`.
- EchoPayload defaults to `"hi"` (root Setup).

## Steps

1. Start LocalRelay with echo DialFunc.
2. Client dials LocalPort, writes payload, reads same bytes.
3. Assert EchoGot equals payload; LocalPort > 0.

## Context

- Proves accept loop + bidirectional copy without real SSH.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioRelayEcho
	req.EchoPayload = "hi"
	return nil
}
```
