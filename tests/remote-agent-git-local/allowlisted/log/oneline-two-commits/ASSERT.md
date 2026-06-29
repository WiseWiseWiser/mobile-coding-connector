## Expected

1. Exit 0.
2. Stdout has exactly two non-empty lines (allow trailing newline).
3. First line contains `Second commit`; second line contains `Initial commit`.

## Side Effects

None.

## Errors

- Wrong order (oldest first) or fewer than two commits shown.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s", resp.ExitCode, resp.Stdout)
	}
	lines := nonEmptyLines(resp.Stdout)
	if len(lines) != 2 {
		t.Fatalf("want 2 oneline log lines, got %d:\n%s", len(lines), resp.Stdout)
	}
	if !strings.Contains(lines[0], "Second commit") {
		t.Fatalf("newest commit first; lines=%v", lines)
	}
	if !strings.Contains(lines[1], "Initial commit") {
		t.Fatalf("second line should be Initial commit; lines=%v", lines)
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
```