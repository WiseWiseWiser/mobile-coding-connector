# Scenario

**Feature**: command mode strips leading user@host before remote argv

```
# dest + command
operator -> sshcmd ["agent@ra", "ls", "-la"]
  -> ModeCommand; Dest="agent@ra"; RemoteArgv=["ls","-la"]
```

## Preconditions

- First token matches strict dest matcher; remaining tokens are remote argv.

## Steps

1. Set Args with dest + multi-token command.
2. Leaf sets Alive session for runner assertion.

## Context

- Dest is documentation-only; runner must not see `agent@ra` in remote argv.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"agent@ra", "ls", "-la"}
	return nil
}
```
