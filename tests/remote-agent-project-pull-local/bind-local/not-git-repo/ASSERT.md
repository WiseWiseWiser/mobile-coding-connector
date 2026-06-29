## Expected

1. Exit code 1.
2. Combined output states local path is not a git repository (or similar).

## Side Effects

None.

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
	if !strings.Contains(combined, "git") {
		t.Fatalf("expected not-a-git-repo message;\n%s", resp.Combined)
	}
}
```