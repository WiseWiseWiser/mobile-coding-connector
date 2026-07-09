# Scenario

**Feature**: dry-run still enforces upload destination guard on non-empty remoteDir

```
# uploads/mirror/existing.txt present -> --dry-run upload -> guard error, serverHome unchanged
pre-seeded file blocks dry-run plan before any would-upload lines
```

## Preconditions

`uploads/mirror/existing.txt` exists on server before upload.

## Steps

1. Pre-seed `uploads/mirror/existing.txt`.
2. Build local tree that would mirror into `uploads/mirror`.
3. Args: `upload --dry-run <localDir> uploads/mirror`.

## Context

REQUIREMENT-DESIGN-upload-download-dry-run.md — dir-rejected/dry-run-guard-fails.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	localRoot := mkLocalWorkDir(t)
	seedRejectLocalTree(t, localRoot)
	req.ServerPreseedFiles = map[string]string{
		"uploads/mirror/existing.txt": seedExistingFileContent,
	}
	setUploadDryRunArgs(t, req, localRoot, "uploads/mirror")
	req.RemoteDir = "uploads/mirror"
	return nil
}
```