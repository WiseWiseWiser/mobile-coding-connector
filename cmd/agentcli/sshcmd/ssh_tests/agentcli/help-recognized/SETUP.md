# Scenario

**Feature**: `ssh --help` is recognized (not unknown command)

```
# help dispatch
agentcli.Run(..., ["ssh", "--help"]) -> nil error
not "unknown command: ssh"
```

## Preconditions

- Scenario: `agentcli-help`.

## Steps

1. Run agentcli with `ssh --help`.
2. Assert AgentcliErr empty and UnknownCommand false.

## Context

- Usage body covered by P1; here only top-level recognition.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioAgentcliHelp
	return nil
}
```
