# Scenario

**Feature**: Authorized client runs remote command; stdout contains needle

```
# remote exec
authorized client -> session.Run("echo hello")
AdhocServer -> exec command -> stdout "hello\n"
```

## Preconditions

- Scenario: `adhoc-auth-accept-command`.
- RemoteCommand default `echo hello`; EchoNeedle `hello`.

## Steps

1. Start Adhoc with authorized key.
2. SSH dial with that key; Run remote command.
3. Capture stdout for Assert.

## Context

- Direct dial (no relay).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioAdhocAuthAcceptCommand
	req.RemoteCommand = "echo hello"
	req.EchoNeedle = "hello"
	return nil
}
```
