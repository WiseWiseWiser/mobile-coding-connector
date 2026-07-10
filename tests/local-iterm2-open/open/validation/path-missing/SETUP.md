# Scenario

**Feature**: missing filesystem path is rejected with 4xx

```
POST {dir: /no/such/path-...} -> 4xx {"error":...}
```

## Preconditions

Path must not exist.

## Steps

1. Set `Dir` under temp to a non-existent child path.
2. Inject-only Open (`UseRealOpenConfig=false`) so handler must validate existence → 4xx.

## Context

REQUIREMENT scenario 4 + locked decision: path missing → **4xx**.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Dir = filepath.Join(t.TempDir(), "does-not-exist-subdir")
	req.UseRealOpenConfig = false
	return nil
}
```
