# Scenario

**Feature**: login with Alive session invokes runner with empty remote argv

```
# happy login
operator -> login, Store.Load -> Alive session
  -> SSHRunner.Run(sess, []) once
```

## Preconditions

- Args empty.
- `Session` is a non-nil Alive session (`aliveSession()`).

## Steps

1. Set Session to Alive fixture.
2. Assert Runner called once with empty remote argv and same session.

## Context

- Interactive login path at the injectable runner boundary.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{}
	req.Session = aliveSession()
	return nil
}
```
