# Scenario

**Feature**: agentcli top-level `ssh` subcommand wiring

```
# dispatch
agentcli.Run(RemoteProfile(), ["ssh", ...])
  -> not "unknown command: ssh"
  -> sshcmd with FileSessionStore + CryptoSSHRunner defaults
```

## Preconditions

- Scenario family: agentcli.
- `case "ssh"` in agentcli.Run may be missing (RED: unknown command).

## Steps

1. Call agentcli.Run in-process with argv starting at `ssh`.
2. Assert error contracts (help nil; no-session tunnel string).

## Context

- Does not capture process stdout (parallel-safe). Help content sealed in P1.
- No live serve required for these leaves.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	return nil
}
```
