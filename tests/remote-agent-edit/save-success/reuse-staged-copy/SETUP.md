# Scenario

**Feature**: an up-to-date staged copy is reused instead of downloaded again

```
# staged copy already hashes to the remote md5 -> skip the GET, edit it directly
staged "old\n" == remote "old\n" -> remote-agent edit -> Skipped download -> Saved
```

## Preconditions

Remote `notes.md` contains `old\n`; the staged path is pre-seeded with the same
bytes (a previous edit of this file).

## Steps

1. Seed `notes.md` via `setEditArgs`; pre-seed the staged copy with identical content.
2. Editor writes `new\n`.
3. Assert exit 0, a `Skipped download` line, **zero** download requests, and the save.

## Context

Optimization leaf — save-success/reuse-staged-copy. Digest equality means the
staged bytes are the remote bytes, so the base md5 and the conflict guarantee are
unchanged while no transfer happens.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StagedPreseed = standardRemoteInitial
	req.EditorWrite = "new\n"
	return nil
}
```
