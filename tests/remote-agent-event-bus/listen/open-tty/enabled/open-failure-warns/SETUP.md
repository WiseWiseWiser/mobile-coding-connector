# Scenario

**Feature**: open failure warns and continues

```
OpenTTY + OpenTTYSession returns error
  -> warning: on stderr; MaxEvents still exit 0
```

## Steps

1. One tty.started with session_id A.
2. OpenTTYFail=true (injected hook returns error).
3. MaxEvents=1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	seedInjectTTY(req, fixtureEventID1, fixtureTS1, fixtureSessionIDA)
	req.OpenTTYFail = true
	req.MaxEvents = 1
	return nil
}
```
