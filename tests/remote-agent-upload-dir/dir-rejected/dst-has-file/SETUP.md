# Scenario

**Feature**: reject directory upload when destination directory contains a file

```
# uploads/mirror/existing.txt present -> upload localDir uploads/mirror -> fail, seed unchanged
pre-seeded file blocks mirror
```

## Preconditions

`uploads/mirror/existing.txt` exists on server before upload.

## Steps

1. Pre-seed `uploads/mirror/existing.txt`.
2. Build local tree that would mirror into `uploads/mirror`.
3. Args: `upload <localDir> uploads/mirror`.

## Context

REQUIREMENT leaf #6 — dir-rejected/dst-has-file.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: destination guard reject via product binaries.
	req.UseCLI = true
	localRoot := mkLocalWorkDir(t)
	seedRejectLocalTree(t, localRoot)
	req.ServerPreseedFiles = map[string]string{
		"uploads/mirror/existing.txt": seedExistingFileContent,
	}
	setUploadArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = "uploads/mirror"
	return nil
}
```