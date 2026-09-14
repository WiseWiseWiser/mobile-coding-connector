# Scenario

**Feature**: trailing-slash remote container nests basename when parent exists

```
pre-create empty parent/
  -> upload ./proj parent/
  -> parent/proj/file.txt
```

## Preconditions

`parent/` exists (empty) on server.

## Steps

1. Create local directory named `proj` with `file.txt`.
2. Pre-seed empty `parent`.
3. Args: `upload <localProjDir> parent/`.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	projDir := filepath.Join(localRoot, "proj")
	writeLocalFile(t, projDir, "file.txt", "proj payload\n", 0644)
	req.ServerPreseedDirs = []string{"parent"}
	setUploadArgs(t, req, projDir, "parent/")
	req.RemoteDir = "parent/proj"
	return nil
}
```
