# Scenario

**Feature**: directory upload succeeds (tar.xz pack → upload → remote apply)

```
localDir + resolved remote dest -> remote-agent upload -> files on server
```

## Preconditions

Destination resolution follows cp -R rules.

## Steps

1. Leaf builds `localDir` tree and sets expected remote paths.
2. Optionally pre-create parent directories via `ServerPreseedDirs`.
3. Assertions expect exit 0 and mirrored/nested paths.

## Context

```go
import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"

	"github.com/xhd2015/doctest/session"
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
	rel = strings.TrimSuffix(rel, "/")
	return filepath.ToSlash(rel)
}

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```
