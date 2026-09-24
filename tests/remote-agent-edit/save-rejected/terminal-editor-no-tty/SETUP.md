# Scenario

**Feature**: a terminal editor without a tty fails fast instead of hanging

```
# --editor=vim with stdin not a tty (agent / piped run) -> Error before launching
serverHome notes.md "old\n" -> remote-agent edit --editor=vim (no tty) -> Error
```

## Preconditions

Remote `notes.md` contains `old\n`. A fake `vim` executable (absolute path) is
installed as the editor and records whether it ran. The run uses the **L3 product
binary**, whose stdin is `/dev/null` (a non-tty in every environment).

## Steps

1. Seed `notes.md` and install the fake `vim` (`FakeTerminalEditor`); the harness
   passes its absolute path as `--editor`, so the guard (not a missing binary)
   decides.
2. Assert exit 1, the `needs a terminal on stdin` error, that the fake `vim`
   never ran, that nothing was staged, and that the remote is untouched.

## Context

Rejection leaf #6 — save-rejected/terminal-editor-no-tty. Agents and scripts run
with piped stdin; launching vim there would hang or corrupt the terminal, so the
guard runs before the download. Needs a subprocess (L3) because an in-process run
cannot present a non-tty stdin without mutating process state.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.UseCLI = true
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.FakeTerminalEditor = "vim"
	return nil
}
```
