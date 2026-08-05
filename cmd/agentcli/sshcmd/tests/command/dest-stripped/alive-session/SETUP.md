# Scenario

**Feature**: dest-stripped command runs with remote ["ls","-la"]

```
# strip then run
operator -> ["agent@ra","ls","-la"] + Alive
  -> Runner.Run(sess, ["ls","-la"])
```

## Preconditions

- Args: `["agent@ra", "ls", "-la"]`.
- Session Alive.

## Steps

1. Set Alive Session.
2. Assert Dest and Runner argv without dest.

## Context

- Covers requirement case: `agent@ra` `ls` `-la`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"agent@ra", "ls", "-la"}
	req.Session = aliveSession()
	return nil
}
```
