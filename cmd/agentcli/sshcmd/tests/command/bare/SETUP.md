# Scenario

**Feature**: bare command token without user@host

```
# bare command
operator -> sshcmd ["ls"]
  -> ModeCommand; Dest=""; RemoteArgv=["ls"]
```

## Preconditions

- First token is not a matching dest (`ls` has no `@`).
- Session state set by leaf.

## Steps

1. Set Args to `["ls"]`.
2. Leaf overlays Session.

## Context

- Equivalent to omitting optional destination.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"ls"}
	return nil
}
```
