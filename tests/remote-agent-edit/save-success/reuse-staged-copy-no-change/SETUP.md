# Scenario

**Feature**: reuse plus an unchanged edit is the zero-transfer path

```
# staged == remote and the editor changes nothing -> no download, no write
staged "old\n" == remote "old\n" -> remote-agent edit (no-op) -> file not changed
```

## Preconditions

Remote `notes.md` contains `old\n`; the staged copy holds the same bytes; the
editor exits 0 without touching it.

## Steps

1. Seed `notes.md`, pre-seed the staged copy identically, set `EditorNoop`.
2. Assert exit 0, `Skipped download`, `file not changed`, and neither a
   `/api/files/download` nor a `/api/files/write` request.

## Context

Optimization leaf — save-success/reuse-staged-copy-no-change. This is the cheapest
repeat run: one check request, one local hash, nothing else.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StagedPreseed = standardRemoteInitial
	req.EditorNoop = true
	return nil
}
```
