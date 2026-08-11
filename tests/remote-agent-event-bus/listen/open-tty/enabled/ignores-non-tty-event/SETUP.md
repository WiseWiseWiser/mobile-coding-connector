# Scenario

**Feature**: only agent.tty.started opens; seatalk does not

```
OpenTTY + inject seatalk.message.received
  -> OpenTTYSession never called; event still printed
```

## Steps

1. One inject seatalk.message.received (with a session_id-like field if any —
   seatalk payload is text-only; open must key off event type).
2. MaxEvents=1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	// Deliberately include session_id in payload: type must still gate open.
	seedInjectSeatalk(req, fixtureEventID1, fixtureTS1,
		`{"text":"hello","session_id":"should-not-open"}`)
	req.MaxEvents = 1
	return nil
}
```
