## Expected

1. Non-zero exit code.
2. Stderr (or Combined) contains `Error:` and indicates the subcommand is
   unknown / invalid (or points at usage).
3. Does not print a successful sessions table for `nosuch`.

## Errors

- Exit 0.
- Silent failure without Error: prefix.

## Exit Code

non-zero

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected non-zero exit for unknown subcommand; stdout:\n%s\nstderr:\n%s",
			resp.Stdout, resp.Stderr)
	}
	combined := resp.Combined
	if combined == "" {
		combined = resp.Stderr + resp.Stdout
	}
	if !strings.Contains(combined, "Error:") && !strings.Contains(strings.ToLower(combined), "error") {
		t.Fatalf("expected Error: on failure; combined:\n%s", combined)
	}
	lower := strings.ToLower(combined)
	if !strings.Contains(lower, "unknown") && !strings.Contains(lower, "usage") &&
		!strings.Contains(lower, "invalid") && !strings.Contains(lower, "nosuch") &&
		!strings.Contains(lower, "unrecognized") {
		// Accept any clear rejection that names the problem domain.
		if !strings.Contains(lower, "agent-run") && !strings.Contains(lower, "subcommand") {
			t.Fatalf("expected clear unknown-subcommand message; combined:\n%s", combined)
		}
	}
}
```
