# Scenario

**Feature**: an editor that rewrites identical bytes is still "not changed"

```
# editor writes the same content as the base -> md5 equal -> file not changed
serverHome notes.md "old\n" -> remote-agent edit -> editor writes "old\n" -> file not changed
```

## Preconditions

Remote `notes.md` contains `old\n`; the editor rewrites exactly `old\n` (a save
without modification, e.g. `:wq` in vim).

## Steps

1. Seed `notes.md` via `setEditArgs`; set `EditorWrite` to the same content.
2. Assert exit 0, `file not changed`, no `Saved`, and untouched remote bytes.

## Context

Happy-path leaf #10 — save-success/reverted-content. Guards the md5 comparison
(rather than mtime or size) as the change signal.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = standardRemoteInitial
	return nil
}
```
