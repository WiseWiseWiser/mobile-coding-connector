# Scenario

**Feature**: event-bus listen --open-tty open-on-event

```
# injectable OpenTTYSession + process-local session_id dedupe
RunEventBusListen(OpenTTY, OpenTTYSession) <- agent.tty.started
  -> open once | warning: | never open
```

## Steps

1. Op=listen; leaves choose OpenTTY on/off and inject event stream.
2. Always install recording OpenTTYSession via InjectOpenHook (or OpenTTY).
3. Prefer DialMode=inject for deterministic multi-event sequences.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "listen"
	if req.DialMode == "" {
		req.DialMode = "inject"
	}
	// Default: install hook so disabled leaves can prove zero calls.
	req.InjectOpenHook = true
	return nil
}
```
