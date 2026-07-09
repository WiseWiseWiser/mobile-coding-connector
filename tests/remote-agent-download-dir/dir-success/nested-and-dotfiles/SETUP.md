# Scenario

**Feature**: directory download mirrors dotfiles and empty subdirectories

```
# remote .hidden, sub/.keep, empty emptydir/ -> ./local-dot
remote tree includes dot paths and emptydir/
```

## Preconditions

Remote destination `uploads/dot-mirror` seeded with dotfiles and empty subdir.

## Steps

1. Seed dot remote tree via `seedDotRemoteTree`.
2. Args: `download uploads/dot-mirror ./local-dot`.

## Context

REQUIREMENT leaf #6 — dir-success/nested-and-dotfiles.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedDotRemoteTree(req)
	setDownloadArgs(t, req, "uploads/dot-mirror", "./local-dot")
	req.LocalDir = localDirRel("uploads/dot-mirror", "./local-dot")
	return nil
}
```