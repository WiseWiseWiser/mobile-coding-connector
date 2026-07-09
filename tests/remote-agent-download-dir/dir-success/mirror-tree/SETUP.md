# Scenario

**Feature**: directory download mirrors remote tree locally

```
# remote a.txt + sub/b.txt -> ./local-mirror -> mirrored local tree
serverHome uploads/mirror -> remote-agent download -> local-mirror/{a.txt,sub/b.txt}
```

## Preconditions

`uploads/mirror` exists on server with standard two-file tree; local destination absent.

## Steps

1. Seed standard remote tree via `seedStandardRemoteTree`.
2. Args: `download uploads/mirror ./local-mirror`.

## Context

REQUIREMENT leaf #2 — dir-success/mirror-tree.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedStandardRemoteTree(req)
	setDownloadArgs(t, req, "uploads/mirror", "./local-mirror")
	req.LocalDir = localDirRel("uploads/mirror", "./local-mirror")
	return nil
}
```