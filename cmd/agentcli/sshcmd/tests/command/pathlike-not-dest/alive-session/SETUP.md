# Scenario

**Feature**: ./a@b runs as remote command token (not stripped dest)

```
# non-dest pathlike
operator -> ["./a@b"] + Alive
  -> ModeCommand; Dest empty; Runner.Run(sess, ["./a@b"])
```

## Preconditions

- Args: `["./a@b"]`.
- Session Alive.
- Strict matcher: `^[^@/\s]+@[^@/\s]+$` does not match.

## Steps

1. Set Alive Session.
2. Assert Dest empty and Runner argv is `["./a@b"]`.

## Context

- Requirement case 7: path-like non-match treated as command.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"./a@b"}
	req.Session = aliveSession()
	return nil
}
```
