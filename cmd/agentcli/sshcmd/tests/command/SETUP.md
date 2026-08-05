# Scenario

**Feature**: client command mode (non-empty remote argv)

```
# command
operator -> sshcmd [dest?] <cmd> [args...]
  -> ModeCommand; RemoteArgv after optional dest strip
  -> SessionStore.Load; Alive gate; SSHRunner.Run(sess, remoteArgv)
```

## Preconditions

- Mode under test is **command**: remote argv non-empty after optional dest strip.
- Leaves set concrete tokens and session state.

## Steps

1. Grouping marks command mode.
2. Child dirs choose argv pattern (bare, dest+cmd, multi-arg, pathlike non-dest).
3. Session leaves set nil vs Alive.

## Context

- OpenSSH-shaped remote command without a dedicated `ssh exec` subcommand.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	return nil
}
```
