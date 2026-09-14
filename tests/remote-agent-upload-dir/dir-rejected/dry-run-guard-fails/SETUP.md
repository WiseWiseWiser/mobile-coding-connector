# Scenario

**Feature**: dry-run + `--no-override` fails preflight before packing

```
pre-seed uploads/apps/srcdir/a.txt
  -> upload --dry-run --no-override ./srcdir uploads/apps
  -> error; server unchanged
```

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
	req.ServerPreseedFiles = map[string]string{
		"uploads/apps/srcdir/a.txt": "seed-existing\n",
	}
	req.ServerPreseedDirs = []string{"uploads/apps", "uploads/apps/srcdir"}
	abs, err := filepath.Abs(src)
	if err != nil {
		t.Fatal(err)
	}
	req.LocalPath = abs
	req.RemotePath = "uploads/apps"
	req.Args = []string{"upload", "--dry-run", "--no-override", abs, "uploads/apps"}
	req.RemoteDir = "uploads/apps/srcdir"
	return nil
}
```
