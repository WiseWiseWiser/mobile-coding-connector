# Scenario

**Feature**: edited staged content replaces the remote file

```
# remote notes.md "old\n" -> staged edit "new\n" -> md5 precondition holds -> Saved
serverHome notes.md -> remote-agent edit notes.md -> notes.md "new\n"
```

## Preconditions

Remote `notes.md` exists with `old\n`; no staged copy yet.

## Steps

1. Seed `notes.md` via `setEditArgs`.
2. Editor writes `new\n` to the staged copy.
3. Assert exit 0, `Saved`, updated remote bytes, staged copy kept.

## Context

Happy-path leaf #1 — save-success/edited-file.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "new\n"
	return nil
}
```
