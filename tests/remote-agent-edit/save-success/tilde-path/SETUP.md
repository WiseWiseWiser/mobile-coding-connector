# Scenario

**Feature**: `~/path` resolves against the server home before staging

```
# edit ~/notes.md -> resolved to <serverHome>/notes.md -> staged <staging>/<serverHome>/notes.md
serverHome notes.md "old\n" -> remote-agent edit ~/notes.md -> Saved
```

## Preconditions

Remote `notes.md` exists in the server home; the CLI argument uses `~/`.

## Steps

1. Seed `notes.md` and pass the argument as `~/notes.md`.
2. Editor writes `home\n`.
3. Assert exit 0, that the staged path is `<staging>/<serverHome>/notes.md`, and
   that the resolved remote path was saved.

## Context

Happy-path leaf #11 — save-success/tilde-path. `ResolveRemoteFilePath` must run
before staging so the same remote file always maps to the same staged path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.FileArg = "~/" + standardRemoteFile
	req.Args = []string{"edit", "~/" + standardRemoteFile}
	req.EditorWrite = "home\n"
	return nil
}
```
