# Scenario

**Feature**: `--editor` values may carry arguments

```
# --editor="<script> --mark" -> argv [script, --mark, staged] -> editor writes staged
serverHome notes.md -> remote-agent edit --editor="script --mark" -> Saved
```

## Preconditions

Remote `notes.md` exists with `old\n`.

## Steps

1. Seed `notes.md` via `setEditArgs`; set `EditorArgsSuffix` so the injected
   `--editor` value is `<script> --mark`.
2. The generated editor takes the **last** argv element as the staged path.
3. Assert exit 0 and the saved content.

## Context

Happy-path leaf #5 — save-success/editor-with-args (spec: `--editor=vim|code|nano`
values may include flags, e.g. `code --wait`).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, standardRemoteFile, standardRemoteInitial)
	req.EditorWrite = "with args\n"
	req.EditorArgsSuffix = "--mark"
	return nil
}
```
