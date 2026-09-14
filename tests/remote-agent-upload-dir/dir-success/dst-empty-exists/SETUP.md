# Scenario

**Feature**: existing remote directory nests basename(localDir)

```
pre-create empty uploads/apps
  -> upload ./srcdir uploads/apps
  -> uploads/apps/srcdir/{a.txt,sub/b.txt}
```

## Preconditions

`uploads/apps` exists on server with zero entries.

## Steps

1. Build local tree under fixed name `srcdir`.
2. Pre-seed empty `uploads/apps`.
3. Args: `upload <srcdir> uploads/apps`.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	src := filepath.Join(localRoot, "srcdir")
	seedStandardLocalTree(t, src)
	req.ServerPreseedDirs = []string{"uploads/apps"}
	setUploadArgs(t, req, src, "uploads/apps")
	req.RemoteDir = "uploads/apps/srcdir"
	return nil
}
```
