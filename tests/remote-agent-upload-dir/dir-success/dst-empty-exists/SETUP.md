# Scenario

**Feature**: directory upload into an empty existing remote directory

```
# pre-create empty uploads/mirror -> same local tree -> mirrored contents
empty remoteDir -> remote-agent upload -> populated mirror tree
```

## Preconditions

`uploads/mirror` exists on server with zero entries.

## Steps

1. Build standard local tree.
2. Pre-seed empty `uploads/mirror` via `ServerPreseedDirs`.
3. Args: `upload <localDir> uploads/mirror`.

## Context

REQUIREMENT leaf #3 — dir-success/dst-empty-exists.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedStandardLocalTree(t, localRoot)
	req.ServerPreseedDirs = []string{"uploads/mirror"}
	setUploadArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = remoteDirRel(localRoot, "uploads/mirror")
	return nil
}
```