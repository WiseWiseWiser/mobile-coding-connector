# Scenario

**Feature**: `edit` refuses to write when the remote changed meanwhile

```
# staged edit -> POST /api/files/write 409 -> nothing written, staged kept
remote file changed by someone else -> remote-agent edit -> Error + conflict recipe
```

## Preconditions

Leaf seeds the remote fixture and makes the editor script change the remote file
(or delete it, or fail) while the staged copy is open.

## Steps

1. Leaf seeds `serverHome` via `setEditArgs` / `setConflictArgs` (root SETUP.md).
2. `Run` stages and opens the editor, which mutates both sides.
3. Assertions expect a non-zero exit, no remote write, and a staged copy that
   still holds the user's edits.

## Context

Rejection leaves: md5 conflict, remote deleted, editor abort, missing editor,
directory target, terminal editor without a tty.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```
