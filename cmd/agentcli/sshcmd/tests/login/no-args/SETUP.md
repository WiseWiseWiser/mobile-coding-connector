# Scenario

**Feature**: login with no positional args

```
# empty argv after ssh
operator -> sshcmd []
  -> ModeLogin; Dest=""; RemoteArgv=[]
```

## Preconditions

- Args: empty slice (no dest, no command).
- Session state set by leaf.

## Steps

1. Set Args to empty.
2. Leaf overlays Session nil or Alive.

## Context

- Default client form when operator omits user@host and command.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{}
	return nil
}
```
