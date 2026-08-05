# Scenario

**Feature**: login with optional user@host destination only

```
# dest strip then login
operator -> sshcmd ["agent@ra"]
  -> ModeLogin; Dest="agent@ra"; RemoteArgv=[]
```

## Preconditions

- Args: single token matching `^[^@/\s]+@[^@/\s]+$`.
- Remote command absent after strip.

## Steps

1. Set dest-only argv shape; leaf sets Alive session.
2. Assert strip + login runner path.

## Context

- Destination is optional documentation for OpenSSH shape; ignored for routing.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"agent@ra"}
	return nil
}
```
