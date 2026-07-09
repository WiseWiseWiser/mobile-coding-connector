# Scenario

**Feature**: directory upload dry-run streams plan without server mutations

```
# local a.txt + sub/b.txt -> --dry-run upload uploads/mirror -> would upload lines, serverHome unchanged
localDir -> remote-agent upload --dry-run -> stdout plan only, no remote bytes
```

## Preconditions

`uploads/mirror` does not exist under `serverHome` before upload.

## Steps

1. Build standard local tree (`a.txt`, `sub/b.txt`).
2. Args: `upload --dry-run <localDir> uploads/mirror`.

## Context

REQUIREMENT-DESIGN-upload-download-dry-run.md — dir-success/dry-run-mirror.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedStandardLocalTree(t, localRoot)
	setUploadDryRunArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = remoteDirRel(localRoot, "uploads/mirror")
	return nil
}
```