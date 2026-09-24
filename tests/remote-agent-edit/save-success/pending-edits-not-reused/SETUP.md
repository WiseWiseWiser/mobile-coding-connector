# Scenario

**Feature**: pending staged edits are never mistaken for an up-to-date base

```
# staged copy holds "mine\n" (e.g. after a conflict) while the remote has "old\n"
staged "mine\n" != remote "old\n" -> remote-agent edit -> Downloading -> staged replaced
```

## Preconditions

Remote `notes.md` contains `old\n`; the staged path holds different bytes
(`mine\n`) from a previous run.

## Steps

1. Seed `notes.md`, pre-seed the staged copy with `mine\n`, set `EditorNoop` so the
   staged bytes after the run are exactly what the download produced.
2. Assert exit 0, a `Downloading` line (no reuse), one download request, and that
   the staged copy now holds the remote content.

## Context

Optimization leaf — save-success/pending-edits-not-reused. Reuse is decided by
digest equality only, so divergent local content always triggers a fresh
download; the proof is that a no-op editor leaves the *remote* bytes in the
staged file.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StagedPreseed = "mine\n"
	req.EditorNoop = true
	return nil
}
```
