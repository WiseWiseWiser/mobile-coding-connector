# Scenario

**Feature**: parent dir with existing sibling subdir still nests basename (success)

```
pre-seed uploads/apps/child/
  -> upload ./srcdir uploads/apps
  -> uploads/apps/srcdir/... (child/ untouched)
```

Note: leaf kept under dir-rejected/ for tree stability; behavior is now success.

```go
import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	src := filepath.Join(localRoot, "srcdir")
	writeLocalFile(t, src, "a.txt", "alpha\n", 0644)
	writeLocalFile(t, src, "sub/b.txt", "bravo\n", 0644)
	req.ServerPreseedDirs = []string{"uploads/apps/child"}
	setUploadArgs(t, req, src, "uploads/apps")
	req.RemoteDir = "uploads/apps/srcdir"
	return nil
}
```
