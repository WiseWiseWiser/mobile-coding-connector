# Scenario

**Feature**: dest-only login strips user@host and runs empty remote argv

```
# strip + login
operator -> ["agent@ra"] + Alive session
  -> Dest stripped; Runner.Run(sess, [])
```

## Preconditions

- Args: `["agent@ra"]`.
- Session Alive.

## Steps

1. Keep dest-only Args; set Alive Session.
2. Assert ModeLogin, Dest, empty remote, one Runner call.

## Context

- Strict matcher accepts `agent@ra`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"agent@ra"}
	req.Session = aliveSession()
	return nil
}
```
