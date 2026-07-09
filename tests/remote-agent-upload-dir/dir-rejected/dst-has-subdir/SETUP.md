# Scenario

**Feature**: reject directory upload when destination directory contains a subdirectory

```
# uploads/mirror/child/ present (even empty) -> upload blocked
any directory entry disqualifies destination
```

## Preconditions

`uploads/mirror/child/` exists on server (empty subdirectory).

## Steps

1. Pre-seed empty `uploads/mirror/child/`.
2. Build local tree and args targeting `uploads/mirror`.

## Context

REQUIREMENT leaf #7 — dir-rejected/dst-has-subdir.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedRejectLocalTree(t, localRoot)
	req.ServerPreseedDirs = []string{"uploads/mirror/child"}
	setUploadArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = "uploads/mirror"
	return nil
}
```