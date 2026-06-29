## Expected

1. Exit code 1.
2. Combined output mentions origin mismatch (case-insensitive `origin` + `mismatch` or equivalent).

## Side Effects

No new `project_bindings` row (file may exist with empty bindings from harness seed).

## Errors

- Exit 0 or missing mismatch diagnostic.

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
	if !strings.Contains(combined, "origin") {
		t.Fatalf("expected origin mismatch message;\n%s", resp.Combined)
	}
	bindings := readConfigBindings(t, resp.RemoteConfigPath)
	for _, b := range bindings {
		if b.LocalPath == req.LocalPath {
			t.Fatalf("unexpected binding written on mismatch: %+v", b)
		}
	}
}
```