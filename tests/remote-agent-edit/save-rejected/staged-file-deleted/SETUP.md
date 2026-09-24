# Scenario

**Feature**: an editor that removes the staged copy uploads nothing

```
# editor deletes the staged file -> CLI cannot hash it -> Error, remote untouched
serverHome notes.md "old\n" -> remote-agent edit (editor rm staged) -> Error
```

## Preconditions

Remote `notes.md` contains `old\n`; the editor removes the staged file before
exiting 0 (e.g. an editor writing to a temp path and losing the original).

## Steps

1. Seed `notes.md` and set `EditorDeleteStaged`.
2. Assert exit 1, the `staged copy … is gone` error, and untouched remote bytes.

## Context

Rejection leaf #7 — save-rejected/staged-file-deleted. A vanished staged copy
must never be interpreted as "delete the remote file" or as "no change".

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorDeleteStaged = true
	return nil
}
```
