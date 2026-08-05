# Scenario

**Feature**: login fails when no active tunnel session

```
# gate
operator -> login, Store.Load -> nil session
  -> error "no active SSH tunnel; run 'remote-agent ssh --serve' first"
  -> Runner not called
```

## Preconditions

- Args empty (login).
- `Session` is nil (store empty / missing).

## Steps

1. Keep Args empty; Session nil.
2. Assert tunnel error and zero Runner calls.

## Context

- Client modes require an active serve session from a prior `--serve`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{}
	req.Session = nil
	return nil
}
```
