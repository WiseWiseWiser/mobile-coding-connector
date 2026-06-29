## Expected

1. Exit code 1.
2. Combined output mentions binding, local path, or non-interactive / TTY requirement.

## Side Effects

Remote remains dirty.

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
	if !strings.Contains(combined, "bind") && !strings.Contains(combined, "local-path") &&
		!strings.Contains(combined, "local path") && !strings.Contains(combined, "tty") &&
		!strings.Contains(combined, "interactive") {
		t.Fatalf("expected binding/TTY hint;\n%s", resp.Combined)
	}
	if strings.TrimSpace(gitPorcelain(t, resp.ProjectDir)) == "" {
		t.Fatalf("remote should still be dirty")
	}
}
```