# Scenario

**Feature**: Alive session runs uname -a via SSHRunner

```
# multi-arg happy path
operator -> ["uname","-a"] + Alive
  -> Runner.Run(sess, ["uname","-a"])
```

## Preconditions

- Args: `["uname", "-a"]`.
- Session Alive.

## Steps

1. Set Alive Session.
2. Assert Runner remote argv exactly `["uname","-a"]`.

## Context

- Multi-token remote command contract.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"uname", "-a"}
	req.Session = aliveSession()
	return nil
}
```
