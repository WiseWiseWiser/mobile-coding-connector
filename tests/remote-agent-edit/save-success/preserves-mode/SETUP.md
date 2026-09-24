# Scenario

**Feature**: the remote file keeps its mode across an edit

```
# remote run.sh 0755 -> edit -> POST /api/files/write preserves mode -> still 0755
serverHome run.sh (0755) -> remote-agent edit run.sh -> run.sh 0755, new bytes
```

## Preconditions

Remote `run.sh` exists with `#!/bin/sh\n` and mode `0755`.

## Steps

1. Seed `run.sh` with mode 0755 (`ServerPreseedModes`).
2. Editor appends a line.
3. Assert exit 0, `Saved`, content updated, and the mode still 0755.

## Context

Happy-path leaf #9 — save-success/preserves-mode. Config and script files are
often executable; a naive write would reset them to 0644.

```go
import (
	"os"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setEditArgs(req, "run.sh", "#!/bin/sh\n")
	req.ServerPreseedModes = map[string]os.FileMode{"run.sh": 0755}
	req.EditorWrite = "#!/bin/sh\necho hi\n"
	return nil
}
```
