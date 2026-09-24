# Scenario

**Feature**: a stale staged copy of the same size never becomes the base version

```
# staged copy from an earlier run has the same byte length but different bytes
stale staged "ZZZZ" + remote "old\n" -> remote-agent edit -> fresh download -> Saved
```

## Preconditions

Remote `notes.md` contains `old\n` (4 bytes). The staged path is pre-seeded with
`ZZZZ` (also 4 bytes) to trigger the download resume/skip path.

## Steps

1. Seed `notes.md` via `setEditArgs`, set `StaleStaged`.
2. Editor writes `new\n`.
3. Assert exit 0, `Saved`, and no conflict — proving the CLI forced a fresh download
   instead of reusing the same-size staged file.

## Context

Regression leaf — save-success/fresh-base. `Client.DownloadFile` returns early when
the local file size equals the remote size, which would make the stale copy the
recorded base and turn the save into a spurious 409.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.StaleStaged = true
	req.EditorWrite = "new\n"
	return nil
}
```
