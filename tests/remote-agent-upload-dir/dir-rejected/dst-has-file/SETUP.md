# Scenario

**Feature**: `--no-override` rejects when nested dest already has conflicting files

```
pre-seed uploads/apps/srcdir/a.txt
  -> upload --no-override ./srcdir uploads/apps
  -> fail; seed unchanged
```

## Preconditions

`uploads/apps/srcdir/a.txt` exists (so effective dest nests and conflicts).

## Steps

1. Pre-seed conflicting file under nested basename path.
2. Build local `srcdir` tree including `a.txt`.
3. Args: `upload --no-override <srcdir> uploads/apps`.

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
	setUploadArgsWithDryRun(t, req, src, "uploads/apps", false)
	// inject --no-override after "upload"
	req.Args = append([]string{"upload", "--no-override"}, req.Args[1:]...)
	req.RemoteDir = "uploads/apps/srcdir"
	return nil
}
```
