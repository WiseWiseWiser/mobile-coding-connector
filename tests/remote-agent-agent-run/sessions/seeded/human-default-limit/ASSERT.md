## Expected

1. Exit code 0.
2. Human table includes columns `SESSION_ID`, `RUNNER`, `STATUS`, `UPDATED`.
3. At most 10 session data rows (default limit).
4. When total > limit, stdout mentions truncation / showing count (e.g. showing
   10 of 12) or equivalent guidance to raise `--limit`.
5. Newest sessions appear (e.g. `sess-00` present); oldest beyond the window may
   be omitted (`sess-11` absent when sorted newest-first with default 10).

## Errors

- Exit non-zero.
- More than 10 session rows.
- Missing column headers.

## Exit Code

0.

```go
import (
	"fmt"
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
	for _, col := range []string{"SESSION_ID", "RUNNER", "STATUS", "UPDATED"} {
		if !strings.Contains(out, col) {
			t.Fatalf("human list missing column %s; stdout:\n%s", col, out)
		}
	}
	// Count data rows by seeded id pattern.
	n := 0
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("sess-%02d", i)
		if strings.Contains(out, id) {
			n++
		}
	}
	if n > 10 {
		t.Fatalf("default limit should show at most 10 sessions, saw %d; stdout:\n%s", n, out)
	}
	if n < 1 {
		t.Fatalf("expected some session rows; stdout:\n%s", out)
	}
	// Truncation note when total exceeds default limit.
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "showing") && !strings.Contains(lower, "limit") &&
		!strings.Contains(out, "10 of 12") && !strings.Contains(out, "10/12") {
		t.Fatalf("expected truncation / --limit guidance when total>limit; stdout:\n%s", out)
	}
	if !strings.Contains(out, "sess-00") {
		t.Fatalf("expected newest sess-00 in default window; stdout:\n%s", out)
	}
}
```
