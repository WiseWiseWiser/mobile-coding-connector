# Scenario

**Feature**: a server without digest reporting still edits correctly

```
# check answers without md5 (old server) -> warn, download, edit as before
legacy check + staged "old\n" == remote "old\n" -> remote-agent edit -> warning + Downloading + Saved
```

## Preconditions

Remote `notes.md` contains `old\n`; the staged copy already matches it (so a
digest-capable server would skip the download). The L2 server strips `md5` from
`/api/files/check` responses.

## Steps

1. Seed `notes.md`, pre-seed an identical staged copy, set `LegacyCheckNoMD5`.
2. Editor writes `new\n`.
3. Assert exit 0, `warning: server did not report the remote digest; downloading
   the full copy` on stderr, a `Downloading` line, one download request, and the save.

## Context

Fallback leaf — save-success/legacy-server-digest. The optimization must degrade
to today's behavior (never to a failure) when the server cannot report a digest,
which is also what a not-yet-upgraded remote looks like.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StagedPreseed = standardRemoteInitial
	req.LegacyCheckNoMD5 = true
	req.EditorWrite = "new\n"
	return nil
}
```
