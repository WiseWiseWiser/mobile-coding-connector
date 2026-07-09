# Scenario

**Feature**: directory upload rejected by destination pre-flight guard

```
# walk not started when remoteDir is occupied or is a file
pre-seeded serverHome destination -> remote-agent upload -> non-zero exit, no partial writes
```

## Preconditions

`remoteDir` exists on the server as a non-empty directory or as a regular file before upload.

## Steps

1. Leaf seeds `serverHome` via `ServerPreseedFiles` / `ServerPreseedDirs`.
2. Leaf builds matching `localDir` and calls `setUploadArgs`.
3. Assertions expect non-zero exit, actionable error text, and unchanged seeded content.

## Context

Guard must fail before any chunked upload bytes are written.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

const (
	seedExistingFileContent = "seed existing\n"
	seedRemoteFileContent   = "remote file blob\n"
)

func seedRejectLocalTree(t *testing.T, localRoot string) {
	t.Helper()
	writeLocalFile(t, localRoot, "incoming.txt", "should not land\n", 0644)
	writeLocalFile(t, localRoot, "nested/incoming.txt", "also blocked\n", 0644)
}

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```