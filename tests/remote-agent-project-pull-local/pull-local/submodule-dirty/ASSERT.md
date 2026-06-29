## Expected

1. Exit code 1.
2. Combined output references submodule dirtiness and path `submod` (or nested path under it).

## Side Effects

Remote submodule remains dirty; no successful worktree summary.

## Errors

- Exit 0.

## Exit Code

1.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "submod") {
		t.Fatalf("expected submodule path in error;\n%s", resp.Combined)
	}
	if !strings.Contains(combined, "submodule") && !strings.Contains(combined, "dirty") {
		t.Fatalf("expected submodule dirty message;\n%s", resp.Combined)
	}
}
```