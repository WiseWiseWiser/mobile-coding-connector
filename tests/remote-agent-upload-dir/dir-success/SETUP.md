# Scenario

**Feature**: directory upload succeeds when destination is missing or empty

```
# walk local tree -> pre-flight guard OK -> fan-out chunked uploads -> mirrored remote tree
localDir + (absent|empty remoteDir) -> remote-agent upload -> files + empty dirs on server
```

## Preconditions

Destination guard accepts missing paths or directories with zero entries (including dot entries).

## Steps

1. Leaf builds `localDir` tree and sets `RemoteDir` to the expected mirror root (serverHome-relative).
2. Optionally pre-create an empty `remoteDir` via `ServerPreseedDirs`.
3. Assertions expect exit 0, directory start/success stdout, and mirrored paths.

## Context

Each child narrows destination state or local tree shape per REQUIREMENT-DESIGN-remote-agent-upload-dir.md.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func seedStandardLocalTree(t *testing.T, localRoot string) {
	t.Helper()
	writeLocalFile(t, localRoot, "a.txt", "alpha\n", 0644)
	writeLocalFile(t, localRoot, "sub/b.txt", "bravo\n", 0644)
}

func remoteDirRel(localPath, remotePath string) string {
	base := filepath.Base(localPath)
	rel := remotePath
	if rel == "" {
		return base
	}
	if strings.HasSuffix(rel, "/") {
		return filepath.ToSlash(filepath.Join(strings.TrimSuffix(rel, "/"), base))
	}
	return filepath.ToSlash(rel)
}

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```