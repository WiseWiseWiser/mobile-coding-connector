# Scenario

**Feature**: Authorized client opens login shell (empty remote command path)

```
# login shell
authorized client -> RequestPty + Shell
client stdin: "echo shell-ok\nexit\n"
stdout contains shell-ok
```

## Preconditions

- Scenario: `adhoc-login-shell`.
- Server treats empty command as shell (sh or system shell).

## Steps

1. Start Adhoc with authorized key.
2. Dial; RequestPty; Shell; drive echo + exit.
3. Assert ShellOK.

## Context

- Exercises shell channel, not session.Run.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioAdhocLoginShell
	return nil
}
```
