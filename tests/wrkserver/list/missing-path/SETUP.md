# Scenario

**Feature**: recorded path missing on disk surfaces as project error

```
# projects.json points at deleted path
ListProjects -> project present with non-empty error
```

## Preconditions

`projects.json` entry for a path that does not exist.

## Steps

1. Register a non-existent absolute path under `WrkHome`.
2. Invoke list.

## Context

REQUIREMENT scenario 4.

```go
import (
	"path/filepath"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Path under temp parent but never created.
	missing := filepath.Join(mkTempDir(t, "wrkserver-missing-parent-*"), "does-not-exist-repo")
	writeProjectsJSON(t, req.WrkHome, []string{missing})
	return nil
}
```
