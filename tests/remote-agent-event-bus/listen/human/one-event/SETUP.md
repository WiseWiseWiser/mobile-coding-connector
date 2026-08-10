# Scenario

**Feature**: one hub event prints human line

```
# real hub WS
RunEventBusListen -> connected server=… -> hub.Publish(ev)
  -> stdout: HH:MM:SS  seatalk.message.received  …
```

## Steps

1. Hub mode; one live seatalk event; MaxEvents=1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenHub(req)
	seedOneLive(req)
	req.MaxEvents = 1
	return nil
}
```
