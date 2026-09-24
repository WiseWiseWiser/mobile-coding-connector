# Scenario

**Feature**: a non-zero editor exit aborts the save

```
# editor exits 3 (e.g. vim :cq) -> nothing uploaded, staged copy kept
serverHome notes.md "old\n" -> remote-agent edit (editor exit 3) -> Error, remote untouched
```

## Preconditions

Remote `notes.md` contains `old\n`.

## Steps

1. Seed `notes.md` via `setEditArgs`; set `EditorExitCode` = 3.
2. Assert exit 1, the `editor exited with status 3` error naming the staged copy,
   untouched remote bytes, and no write attempt.

## Context

Rejection leaf #3 — save-rejected/editor-nonzero. An abort (`:cq`, failed GUI
close) must never upload, even when the staged content differs.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "should not be saved\n"
	req.EditorExitCode = 3
	return nil
}
```
