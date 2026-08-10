## Expected

1. Exit code 0.
2. Human empty list: either a clear empty-friendly message **or** a table header
   (`SESSION_ID` / `RUNNER` / `STATUS` / `UPDATED`) with no data rows.
3. Must not invent session ids (no `sess-` rows from absent seeds).
4. Stdout is non-empty (not a silent success).

## Errors

- Non-zero exit.
- Silent empty stdout.
- Fabricated session rows.

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
		t.Fatalf("empty list should print a clear empty message or table header, not silent stdout")
	}
	// Must not show seeded-style session ids when store is empty.
	if strings.Contains(out, "sess-") {
		t.Fatalf("empty list leaked session ids; stdout:\n%s", out)
	}
	lower := strings.ToLower(out)
	hasHeader := strings.Contains(out, "SESSION_ID") || strings.Contains(lower, "session")
	hasEmptyMsg := strings.Contains(lower, "no ") || strings.Contains(lower, "empty") ||
		strings.Contains(lower, "0 session")
	if !hasHeader && !hasEmptyMsg {
		// Accept local agent-run style: tabwriter header only.
		if !strings.Contains(out, "RUNNER") && !strings.Contains(out, "STATUS") {
			t.Fatalf("expected empty-friendly message or SESSION_ID table; stdout:\n%s", out)
		}
	}
}
```
