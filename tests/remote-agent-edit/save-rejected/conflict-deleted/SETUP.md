# Scenario

**Feature**: a remote deletion during the edit rejects the save (no resurrection)

```
# remote notes.md removed while the staged copy holds "mine\n"
serverHome notes.md "old\n" -> edit -> remote deleted -> 409 (+ "was deleted")
```

## Preconditions

Remote `notes.md` contains `old\n` at download time; the editor removes it.

## Steps

1. Seed `notes.md`, set the editor to write `mine\n` to the staged copy and
   delete the remote file (`RemoteDelete`).
2. Assert exit 1, the conflict recipe mentioning the deletion, and that the
   remote file is still missing.

## Context

Rejection leaf #2 — save-rejected/conflict-deleted. A missing file counts as
empty content, so the recorded md5 no longer matches and the save is refused
instead of silently recreating the file.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "mine\n"
	req.RemoteDelete = []string{standardRemoteFile}
	return nil
}
```
