# Scenario

**Feature**: Raw bytes round-trip through WS tunnel splice

```
# echo backend (not Adhoc SSH)
Manager.BackendDial -> TCP echo
Client.SSHTunnelDial -> write payload -> read same bytes
```

## Preconditions

- Scenario: `tunnel-binary-echo`.
- EchoPayload default `p4-tunnel-hi`.

## Steps

1. Start TCP echo; inject as Manager.BackendDial.
2. CreateSession; SSHTunnelDial; write/read payload.
3. Assert EchoRead contains/equals EchoWrote.

## Context

- Isolates tunnel transport from SSH framing (requirement optional raw path).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioTunnelBinaryEcho
	req.EchoPayload = "p4-tunnel-hi"
	return nil
}
```
