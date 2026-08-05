# Scenario

**Feature**: path-like token with @ is not a destination under strict matcher

```
# strict matcher
operator -> sshcmd ["./a@b"]
  -> ModeCommand; Dest=""; RemoteArgv=["./a@b"]
  # ./a@b fails ^[^@/\s]+@[^@/\s]+$ because of '/'
```

## Preconditions

- Token `./a@b` contains `/` and must **not** be stripped as dest.

## Steps

1. Set Args to `["./a@b"]`.
2. Leaf sets Alive session; Assert command not login/dest.

## Context

- Prevents treating path arguments as OpenSSH destinations.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"./a@b"}
	return nil
}
```
