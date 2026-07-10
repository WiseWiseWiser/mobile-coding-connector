# Scenario

**Feature**: dir pointing at a file is rejected with 4xx

```
POST {dir: /tmp/file} -> 4xx {"error":...}
# handler validates IsDir before Open (prefer); Open must not be required for 4xx
```

## Preconditions

Create a regular file path.

## Steps

1. Write a temp file; set `Dir` to that path.
2. Inject Open records calls; validation should fail as 4xx without needing live iTerm.
   `UseRealOpenConfig` may be false so failures come from handler validation.

## Context

REQUIREMENT scenario 4 + locked decision: missing/non-dir → **4xx**.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	req.Dir = file
	// Handler should validate before Open; inject-only is enough.
	req.UseRealOpenConfig = false
	return nil
}
```
