# Scenario

**Feature**: a rejected save is a no-op when the remote already has the same bytes

```
# remote changed to exactly the staged content -> 409 -> "already matches", exit 0
remote notes.md "old\n" -> editor writes "same\n" to staged AND remote -> nothing written
```

## Preconditions

Remote `notes.md` contains `old\n`. The editor writes `same\n` to the staged copy
and to the remote file, so the remote changed — but to the same bytes the user saved.

## Steps

1. Seed `notes.md` via `setEditArgs`; set `EditorWrite` and `RemoteWrite` to `same\n`.
2. Assert exit 0, an `already matches` line, no `Saved`, and remote content `same\n`.

## Context

Happy-path leaf #6 — save-success/converged-md5. The server stays strict (409),
but there is nothing to resolve when both sides converged, so the CLI reports
success instead of a conflict recipe.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setConflictArgs(req, standardRemoteFile, standardRemoteInitial, "same\n", "same\n")
	return nil
}
```
