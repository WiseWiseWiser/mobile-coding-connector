# Scenario

**Feature**: `ssh ls` without active serve → tunnel error (not unknown command)

```
# client gate via CLI defaults
agentcli.Run(..., ["ssh", "ls"])
  -> error contains "no active SSH tunnel; run 'remote-agent ssh --serve' first"
  -> not "unknown command: ssh"
  -> not only "session store not configured" without tunnel message
```

## Preconditions

- Scenario: `agentcli-no-session`.
- Default FileSessionStore has no Alive session for default profile.

## Steps

1. Run agentcli with `ssh ls`.
2. Assert error contains tunnel message; UnknownCommand false.

## Context

- Wires real store so missing session hits P1 gate string.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioAgentcliNoSession
	return nil
}
```
