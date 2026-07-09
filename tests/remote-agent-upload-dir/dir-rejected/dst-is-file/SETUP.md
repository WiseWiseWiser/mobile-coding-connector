# Scenario

**Feature**: reject directory upload when destination path is a regular file

```
# uploads/mirror is a file -> upload localDir uploads/mirror -> fail, blob unchanged
destination path occupied by non-directory
```

## Preconditions

`uploads/mirror` exists as a regular file on the server.

## Steps

1. Pre-seed file `uploads/mirror` with known content.
2. Build local tree and args targeting `uploads/mirror`.

## Context

REQUIREMENT leaf #8 — dir-rejected/dst-is-file.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedRejectLocalTree(t, localRoot)
	req.ServerPreseedFiles = map[string]string{
		"uploads/mirror": seedRemoteFileContent,
	}
	setUploadArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = "uploads/mirror"
	return nil
}
```