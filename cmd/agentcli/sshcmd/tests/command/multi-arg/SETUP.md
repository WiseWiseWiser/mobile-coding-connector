# Scenario

**Feature**: multi-argument remote command without dest

```
# multi-arg command
operator -> sshcmd ["uname", "-a"]
  -> ModeCommand; RemoteArgv=["uname","-a"]
```

## Preconditions

- No dest token; two remote tokens.

## Steps

1. Set Args to `uname -a`.
2. Leaf sets Alive for runner check.

## Context

- Requirement case: Command `uname -a`, Alive session.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"uname", "-a"}
	return nil
}
```
