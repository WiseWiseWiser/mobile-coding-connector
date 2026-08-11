# Scenario

**Feature**: process-local dedupe by session_id

```
# three tty.started: A, A, B
OpenTTY -> OpenTTYSession(A) once then OpenTTYSession(B) once
```

## Steps

1. Inject three agent.tty.started: session A, A again, then B.
2. MaxEvents=3 (all printed).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	seedInjectTTY(req, fixtureEventID1, fixtureTS1, fixtureSessionIDA)
	seedInjectTTY(req, fixtureEventID2, fixtureTS2, fixtureSessionIDA)
	seedInjectTTY(req, fixtureEventID3, fixtureTS3, fixtureSessionIDB)
	req.MaxEvents = 3
	return nil
}
```
