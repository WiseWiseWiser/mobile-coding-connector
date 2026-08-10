## Expected

1. Exit code 0.
2. Stdout is help for `agent-run sessions` list mode.
3. Documents `--json` and `--limit` (list flags).

## Errors

- Exit non-zero.
- Help omits `--json` or `--limit`.

## Exit Code

0.

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
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty sessions help")
	}
	if !strings.Contains(out, "--json") {
		t.Fatalf("sessions help must document --json; stdout:\n%s", out)
	}
	if !strings.Contains(out, "--limit") {
		t.Fatalf("sessions help must document --limit; stdout:\n%s", out)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "session") {
		t.Fatalf("sessions help should mention sessions; stdout:\n%s", out)
	}
}
```
