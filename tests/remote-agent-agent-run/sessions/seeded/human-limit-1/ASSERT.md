## Expected

1. Exit code 0.
2. Exactly one of the seeded session ids appears as a data row (newest:
   `sess-new` preferred when sorted newest-first).
3. Stdout indicates truncation (showing 1 of 3 or `--limit` guidance).
4. Column headers present.

## Errors

- Multiple session ids printed.
- Exit non-zero.

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
	if !strings.Contains(out, "SESSION_ID") {
		t.Fatalf("missing SESSION_ID header; stdout:\n%s", out)
	}
	ids := []string{"sess-new", "sess-mid", "sess-old"}
	found := 0
	var which []string
	for _, id := range ids {
		if strings.Contains(out, id) {
			found++
			which = append(which, id)
		}
	}
	if found != 1 {
		t.Fatalf("--limit 1 should show exactly 1 session id, got %d %v; stdout:\n%s", found, which, out)
	}
	// Newest-first → sess-new.
	if which[0] != "sess-new" {
		t.Fatalf("expected newest sess-new under --limit 1, got %v; stdout:\n%s", which, out)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "showing") && !strings.Contains(out, "1 of 3") &&
		!strings.Contains(lower, "limit") {
		t.Fatalf("expected truncation note for limit 1 of 3; stdout:\n%s", out)
	}
}
```
