# Scenario

**Feature**: without --open-tty never opens

```
OpenTTY=false + agent.tty.started{session_id=A}
  -> log only; OpenTTYSession never called
```

## Steps

1. One inject agent.tty.started with session_id A.
2. MaxEvents=1; OpenTTY remains false.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	seedInjectTTY(req, fixtureEventID1, fixtureTS1, fixtureSessionIDA)
	req.MaxEvents = 1
	return nil
}
```
