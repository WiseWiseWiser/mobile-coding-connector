# Scenario

**Feature**: `--remember-flags` persists the editor for later runs

```
# run 1: --editor=<script> --remember-flags -> config saved
# run 2: edit without --editor -> "Using remembered flags" -> remembered script runs
serverHome notes.md -> edit (remember) -> edit (reuse) -> notes.md "second\n"
```

## Preconditions

Remote `notes.md` contains `old\n`; the CLI config dir starts empty.

## Steps

1. Run 1: `edit notes.md` with `--remember-flags` and the generated `--editor`;
   the editor writes `first\n` on its first invocation.
2. Run 2: `edit notes.md` with no editor flag; the remembered script is used and
   writes `second\n` on its second invocation.
3. Assert run 1 exit 0 and `Saved`, run 2 exit 0 with `Using remembered flags`
   and `Saved`, and remote content `second\n`.

## Context

Happy-path leaf #8 — save-success/remember-flags. Proves the remembered flag is
replayed (a fallback to `vim` would fail the terminal guard in this environment)
and that explicit flags are not required on the second run.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "first\n"
	req.EditorWriteSecond = "second\n"
	req.RememberFlags = true
	req.SecondArgs = []string{"edit", standardRemoteFile}
	return nil
}
```
