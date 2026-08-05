# Scenario

**Feature**: client login mode (no remote command)

```
# login
operator -> sshcmd [] | [user@host]
  -> ModeLogin; RemoteArgv empty; dest stripped when matched
  -> SessionStore.Load; require Alive else tunnel error
  -> SSHRunner.Run(sess, [])
```

## Preconditions

- Mode under test is **login**: empty remote argv after optional dest strip.
- Session presence is set by deeper leaves (`Session` nil vs Alive).

## Steps

1. Grouping marks login; child dirs choose argv shape and session state.
2. Leaves set Args / Session; Assert gate or Runner empty remote.

## Context

- Interactive login shell path (P1: runner called with empty remote argv).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	// Child leaves set Args and Session.
	return nil
}
```
