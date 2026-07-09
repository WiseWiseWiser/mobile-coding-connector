# Scenario

**Feature**: directory download succeeds and mirrors remote tree locally

```
# recursive BrowseDir -> per-file GET downloads -> local mirror tree
serverHome remoteDir + (fresh|partial localDir) -> remote-agent download -> files + empty dirs locally
```

## Preconditions

Directory destination may be absent or partially populated (resume leaves pre-seed local files).

## Steps

1. Leaf seeds `serverHome` remote tree and sets `LocalDir` to the expected mirror root.
2. Resume leaves optionally pre-create local files via `LocalPreseedFiles` / `LocalPreseedDirs`.
3. Assertions expect exit 0, directory streaming stdout, and mirrored local paths.

## Context

Each child narrows remote tree shape or resume state per REQUIREMENT-DESIGN-remote-agent-download-dir.md.

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func seedStandardRemoteTree(req *Request) {
	req.ServerPreseedFiles = map[string]string{
		"uploads/mirror/a.txt":     "alpha\n",
		"uploads/mirror/sub/b.txt": "bravo\n",
	}
}

func seedDotRemoteTree(req *Request) {
	req.ServerPreseedFiles = map[string]string{
		"uploads/dot-mirror/.hidden":     "dotfile\n",
		"uploads/dot-mirror/sub/.keep":   "",
	}
	req.ServerPreseedDirs = []string{"uploads/dot-mirror/emptydir"}
}

func localDirRel(remotePath, localPath string) string {
	base := filepath.Base(strings.TrimSuffix(filepath.ToSlash(remotePath), "/"))
	rel := localPath
	if rel == "" {
		return base
	}
	if strings.HasSuffix(rel, "/") || strings.HasSuffix(rel, string(filepath.Separator)) {
		return filepath.ToSlash(filepath.Join(strings.TrimSuffix(filepath.ToSlash(rel), "/"), base))
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