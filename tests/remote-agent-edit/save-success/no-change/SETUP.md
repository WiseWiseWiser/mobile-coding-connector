# Scenario

**Feature**: an editor session that changes nothing reports `file not changed`

```
# staged copy unchanged -> md5 equal -> no request, no remote write
serverHome notes.md -> remote-agent edit (no-op editor) -> file not changed
```

## Preconditions

Remote `notes.md` exists with `old\n`; the editor exits 0 without touching the staged copy.

## Steps

1. Seed `notes.md` via `setEditArgs`.
2. Set `EditorNoop` so the generated editor only exits 0.
3. Assert exit 0, `file not changed`, untouched remote bytes, no `Saved`.

## Context

Happy-path leaf #3 — save-success/no-change (spec: "if nothing changed, show user: file not changed").

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorNoop = true
	return nil
}
```
