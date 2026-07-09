# Scenario

**Feature**: single-file upload unchanged

```
# copy hello.txt -> upload uploads/hello.txt -> remote bytes match
local hello.txt -> remote-agent upload -> uploads/hello.txt on server
```

## Preconditions

No remote pre-seed; destination path absent on server.

## Steps

1. Copy `testdata/hello.txt` into a temp local file.
2. Args: `upload <local> uploads/hello.txt`.

## Context

REQUIREMENT leaf #1 — file-regression/single-file.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	localDir := mkLocalWorkDir(t)
	localFile := filepath.Join(localDir, "hello.txt")
	copyFixture(t, "testdata/hello.txt", localFile)
	setUploadArgs(t, req, localFile, "uploads/hello.txt")
	req.RemoteDir = "uploads"
	return nil
}
```