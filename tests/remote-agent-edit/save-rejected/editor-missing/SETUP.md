# Scenario

**Feature**: a missing editor binary fails before staging anything

```
# --editor=definitely-not-an-editor-xyz -> LookPath fails -> Error, no download
serverHome notes.md "old\n" -> remote-agent edit --editor=<missing> -> Error
```

## Preconditions

Remote `notes.md` contains `old\n`; the editor binary does not exist.

## Steps

1. Seed `notes.md` and pass `--editor definitely-not-an-editor-xyz` in `Args`.
2. Assert exit 1, the `not found in PATH` error naming the editor, untouched
   remote bytes, and no staged copy.

## Context

Rejection leaf #4 — save-rejected/editor-missing. The editor is validated before
the download, so a typo costs nothing (no staging, no transfer).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.Args = []string{"edit", standardRemoteFile, "--editor", "definitely-not-an-editor-xyz"}
	return nil
}
```
