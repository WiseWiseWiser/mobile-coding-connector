# Scenario

**Feature**: a remote file that does not exist is edited from an empty staged copy

```
# remote path absent -> staged empty file -> editor writes content -> Created
serverHome (no notes/todo.md) -> remote-agent edit notes/todo.md -> Created notes/todo.md
```

## Preconditions

`notes/todo.md` (and its parent directory) does not exist on the server.

## Steps

1. Point the CLI at the missing path.
2. Editor writes `buy milk\n` to the staged copy.
3. Assert exit 0, `Created`, remote file created with the content and parent dir.

## Context

Happy-path leaf #2 — save-success/new-file ("if <file> does not exist, treat it as empty").

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditNewFileArgs(req, "notes/todo.md")
	req.EditorWrite = "buy milk\n"
	return nil
}
```
