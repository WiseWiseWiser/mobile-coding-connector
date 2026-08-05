# Scenario

**Feature**: bare command with Alive session invokes runner with ["ls"]

```
# happy bare command
operator -> ["ls"] + Alive
  -> SSHRunner.Run(sess, ["ls"])
```

## Preconditions

- Args: `["ls"]`.
- Session Alive.

## Steps

1. Set Alive Session.
2. Assert Runner remote argv `["ls"]`.

## Context

- Single-token remote command.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"ls"}
	req.Session = aliveSession()
	return nil
}
```
