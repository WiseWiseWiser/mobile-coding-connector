# Scenario

**Feature**: open once on agent.tty.started with session_id

```
OpenTTY + inject agent.tty.started{session_id=A}
  -> OpenTTYSession(A) once; event still printed
```

## Steps

1. One inject event: agent.tty.started with session_id=fixtureSessionIDA.
2. MaxEvents=1.

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
