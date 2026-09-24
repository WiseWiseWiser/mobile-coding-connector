# Scenario

**Feature**: a directory target is rejected before staging

```
# remote path is a directory -> Error, no download, no staged copy
serverHome adir/ (directory) -> remote-agent edit adir -> Error "is a directory"
```

## Preconditions

`adir` exists on the server as a directory.

## Steps

1. Preseed the directory (`ServerPreseedDirs`) and point the CLI at it.
2. Assert exit 1, the `is a directory` error, and no staged copy.

## Context

Rejection leaf #5 — save-rejected/remote-is-dir. `edit` handles a single file
only; directories are refused right after the `CheckPath` probe.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FileArg = "adir"
	req.RemoteRel = "adir"
	req.ServerPreseedDirs = []string{"adir"}
	req.Args = []string{"edit", "adir"}
	return nil
}
```
