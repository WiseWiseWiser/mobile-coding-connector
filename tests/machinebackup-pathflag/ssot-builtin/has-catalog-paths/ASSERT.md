## Expected

1. No Run error.
2. `MissingPaths` is empty (HasPath true).
3. All pathflag catalog rules and `**(binary)` appear in BuiltinExclusionConfig.

## Side Effects

- None.

## Errors

- Any missing path fails.

## Exit Code

- N/A

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(resp.MissingPaths) > 0 {
		t.Fatalf("BuiltinExclusionConfig missing paths: %s", strings.Join(resp.MissingPaths, ", "))
	}
	if !resp.HasPath {
		t.Fatal("HasPath=false, want complete catalog path set")
	}
}
```
