# Scenario

**Feature**: directory upload streams incremental stdout with per-item index and overall rollup

```
# multi-file localDir -> chunked uploads with onProgress -> [N/M] + overall lines before summary
local a.txt + sub/b.txt -> remote-agent upload -> stdout streams progress then Upload complete
```

## Preconditions

Remote destination `uploads/stream-mirror` absent.

## Steps

1. Build standard local tree (`a.txt`, `sub/b.txt`).
2. Args: `upload <localDir> uploads/stream-mirror`.

## Context

REQUIREMENT-DESIGN-upload-streaming-progress.md leaf #1 — dir-success/streams-progress.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedStandardLocalTree(t, localRoot)
	setUploadArgs(t, req, localRoot, "uploads/stream-mirror")
	req.RemoteDir = remoteDirRel(localRoot, "uploads/stream-mirror")
	return nil
}
```