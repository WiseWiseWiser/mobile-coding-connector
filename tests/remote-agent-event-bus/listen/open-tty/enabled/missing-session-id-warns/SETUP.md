# Scenario

**Feature**: missing session_id warns and skips open

```
OpenTTY + agent.tty.started without session_id
  -> warning: on stderr; OpenTTYSession never called; still exit 0
```

## Steps

1. Inject one agent.tty.started with empty session_id (payload lacks it).
2. MaxEvents=1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	// sessionID "" → payload without session_id key value usable for open
	seedInjectTTY(req, fixtureEventID1, fixtureTS1, "")
	req.MaxEvents = 1
	return nil
}
```
