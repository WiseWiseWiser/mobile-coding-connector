## Expected

1. ExitCode 0 (command exists and succeeds).
2. Combined contains `No bookmarks` (preferred) OR clearly empty root listing without error.

## Errors

- Unknown subcommand; non-zero without empty message.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	out := resp.Combined
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, out, resp.ErrMsg)
	}
	low := strings.ToLower(out)
	if !strings.Contains(low, "no bookmarks") && !strings.Contains(low, "bookmarks") {
		t.Fatalf("expected empty-list human message; out=%q", out)
	}
	// Prefer explicit No bookmarks copy from requirement
	if !strings.Contains(out, "No bookmarks") {
		// allow soft pass only if stdout is non-empty and no Error
		if strings.Contains(low, "error") {
			t.Fatalf("error on empty list: %q", out)
		}
		t.Fatalf("want exact-ish empty copy including 'No bookmarks'; got %q", out)
	}
}
```
