## Expected

1. Exit code 0 (keep listening / MaxEvents stop).
2. Event still printed.
3. Stderr (prefer) contains `warning:` about missing/empty session_id.
4. OpenTTYSession never called.

## Errors

- Open called with empty or junk id.
- Silent skip with no warning.
- Hard failure exit.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, fixtureTypeTTY) {
		t.Fatalf("expected event still printed; stdout:\n%s", resp.Stdout)
	}
	warnText := resp.Stderr
	if !strings.Contains(warnText, "warning:") {
		warnText = resp.Combined
	}
	if !strings.Contains(warnText, "warning:") {
		t.Fatalf("expected warning: for missing session_id; stderr:\n%s\nstdout:\n%s",
			resp.Stderr, resp.Stdout)
	}
	// Product should mention session (session_id / session) in the warning text.
	if !strings.Contains(strings.ToLower(warnText), "session") {
		t.Fatalf("warning should mention session_id/session; warn text:\n%s", warnText)
	}
	if len(resp.OpenTTYSessionIDs) != 0 {
		t.Fatalf("OpenTTYSession must not be called; got %v", resp.OpenTTYSessionIDs)
	}
}
```
