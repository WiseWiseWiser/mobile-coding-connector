# Scenario

**Feature**: warning on disconnect then retry

```
DialWS drop-once -> first event -> EOF -> warning: reconnect -> second event
```

## Steps

1. Inject two seatalk-ish events; MaxEvents=2; drop-once dialer.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "listen"
	req.DialMode = "drop-once"
	req.JSON = false
	req.InjectEvents = []EventSeed{
		{
			ID:      fixtureEventID1,
			TS:      fixtureTS1,
			Source:  fixtureSourceBot,
			Type:    fixtureTypeSeatalk,
			Payload: fixturePayload1,
		},
		{
			ID:      fixtureEventID2,
			TS:      fixtureTS2,
			Source:  fixtureSourceBot,
			Type:    fixtureTypeSeatalk,
			Payload: fixturePayload2,
		},
	}
	req.MaxEvents = 2
	return nil
}
```
