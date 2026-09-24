# Scenario

**Feature**: an existing but empty remote file is edited, not recreated

```
# remote empty.md exists (0 bytes) -> staged empty -> editor writes -> Saved (not Created)
serverHome empty.md "" -> remote-agent edit empty.md -> empty.md "filled\n"
```

## Preconditions

Remote `empty.md` exists with zero bytes.

## Steps

1. Seed `empty.md` with empty content.
2. Editor writes `filled\n`.
3. Assert exit 0, `Saved` (never `Created`), and the new remote content.

## Context

Happy-path leaf #12 — save-success/remote-empty-file. An empty file and a missing
file share the same md5, so the write path must still distinguish them for the
`Created` flag and must not treat the empty base as "no precondition".

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, "empty.md", "")
	req.EditorWrite = "filled\n"
	return nil
}
```
