# Scenario

**Feature**: event-bus listen streaming core

```
RunEventBusListen(stdout,stderr,opts) -> connected + event lines
  # hub WS or injectable DialWS
```

## Steps

1. Op=listen; leaves set DialMode, events, JSON/Types/Replay.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "listen"
	if req.DialMode == "" {
		req.DialMode = "hub"
	}
	return nil
}
```
