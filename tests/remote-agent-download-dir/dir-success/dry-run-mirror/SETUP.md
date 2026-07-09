# Scenario

**Feature**: directory download dry-run streams plan without local writes

```
# remote a.txt + sub/b.txt -> --dry-run download -> would download lines, no local files
serverHome uploads/mirror -> remote-agent download --dry-run -> stdout plan only
```

## Preconditions

`uploads/mirror` exists on server with standard two-file tree; local destination absent.

## Steps

1. Seed standard remote tree via `seedStandardRemoteTree`.
2. Args: `download --dry-run uploads/mirror ./local-mirror`.

## Context

REQUIREMENT-DESIGN-upload-download-dry-run.md — dir-success/dry-run-mirror.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedStandardRemoteTree(req)
	setDownloadDryRunArgs(t, req, "uploads/mirror", "./local-mirror")
	req.LocalDir = localDirRel("uploads/mirror", "./local-mirror")
	return nil
}
```