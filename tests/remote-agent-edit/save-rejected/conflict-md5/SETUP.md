# Scenario

**Feature**: a remote change during the edit rejects the save

```
# remote notes.md changed to "theirs\n" while the staged copy holds "mine\n"
serverHome notes.md "old\n" -> edit -> 409 -> nothing written + conflict recipe
```

## Preconditions

Remote `notes.md` contains `old\n` when the CLI downloads it. The editor writes
`mine\n` to the staged copy and `theirs\n` to the remote file.

## Steps

1. Seed `notes.md` and make the editor change both sides (`setConflictArgs`).
2. Assert exit 1, the conflict recipe on stderr, remote content `theirs\n`
   (unchanged by the CLI), and the staged copy still holding `mine\n`.

## Context

Rejection leaf #1 — save-rejected/conflict-md5 (spec: "if md5 changed, fail the
save, and ask user to resolve the conflict").

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setConflictArgs(req, standardRemoteFile, standardRemoteInitial, "mine\n", "theirs\n")
	return nil
}
```
