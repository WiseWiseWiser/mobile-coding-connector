# Scenario

**Feature**: directory upload mirrors dotfiles and empty subdirectories

```
# local .hidden, sub/.keep, empty emptydir/ -> uploads/dot-mirror
remote tree includes dot paths and emptydir/
```

## Preconditions

Remote destination `uploads/dot-mirror` absent.

## Steps

1. Seed local tree with dotfile, nested dotfile, and empty directory.
2. Args: `upload <localDir> uploads/dot-mirror`.

## Context

REQUIREMENT leaf #4 — dir-success/nested-and-dotfiles.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	writeLocalFile(t, localRoot, ".hidden", "dotfile\n", 0644)
	writeLocalFile(t, localRoot, "sub/.keep", "", 0644)
	writeLocalDir(t, localRoot, "emptydir")
	setUploadArgs(t, req, localRoot, "uploads/dot-mirror")
	req.RemoteDir = remoteDirRel(localRoot, "uploads/dot-mirror")
	return nil
}
```