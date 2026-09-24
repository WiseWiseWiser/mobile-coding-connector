# Scenario

**Feature**: a failed digest probe falls back to a plain check and download

```
# check with md5:true -> 500 -> warning + plain check -> download -> edit succeeds
digest probe 500 + remote "old\n" -> remote-agent edit -> warning + Downloading + Saved
```

## Preconditions

Remote `notes.md` contains `old\n`; the staged copy already matches it. The L2
server fails `/api/files/check` requests that ask for a digest (as a proxy timeout
or hashing error would), while plain check requests keep working.

## Steps

1. Seed `notes.md`, pre-seed an identical staged copy, set `DigestProbeStatus` to 500.
2. Editor writes `new\n`.
3. Assert exit 0, a `warning: remote digest unavailable (…)` on stderr, a
   `Downloading` line, two check requests (probe + plain fallback), and the save.

## Context

Fallback leaf — save-success/digest-probe-fails. Guards the decision that the
optimization can never turn a working edit into a failure: the probe is retried
without the digest and the run continues.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StagedPreseed = standardRemoteInitial
	req.DigestProbeStatus = 500
	req.EditorWrite = "new\n"
	return nil
}
```
