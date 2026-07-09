# Scenario

**Feature**: trailing-slash remote path appends basename(localDir)

```
# local proj/ with file.txt -> upload parent/ -> parent/proj/file.txt
basename rule + trailing slash -> contents under parent/proj/
```

## Preconditions

`parent/` absent on server before upload.

## Steps

1. Create local directory named `proj` with `file.txt`.
2. Args: `upload <localProjDir> parent/`.

## Context

REQUIREMENT leaf #5 — dir-success/trailing-slash-dst.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	projDir := filepath.Join(localRoot, "proj")
	writeLocalFile(t, projDir, "file.txt", "proj payload\n", 0644)
	setUploadArgs(t, req, projDir, "parent/")
	req.RemoteDir = remoteDirRel(projDir, "parent/")
	return nil
}
```