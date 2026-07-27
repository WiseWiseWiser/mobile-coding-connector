# Scenario

**Feature**: directory upload when remote destination is absent

```
# local a.txt + sub/b.txt -> uploads/mirror (missing) -> mirrored tree created
localDir -> remote-agent upload uploads/mirror -> uploads/mirror/{a.txt,sub/b.txt}
```

## Preconditions

`uploads/mirror` does not exist under `serverHome` before upload.

## Steps

1. Build standard local tree (`a.txt`, `sub/b.txt`).
2. Args: `upload <localDir> uploads/mirror`.

## Context

REQUIREMENT leaf #2 — dir-success/dst-not-exists.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedStandardLocalTree(t, localRoot)
	setUploadArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = remoteDirRel(localRoot, "uploads/mirror")
	return nil
}
```