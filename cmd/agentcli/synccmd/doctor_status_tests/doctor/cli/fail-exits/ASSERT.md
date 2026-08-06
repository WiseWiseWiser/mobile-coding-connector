## Expected

1. Harness err nil.
2. `RunErr` non-empty (CLI returns error when doctor fails).
3. Stdout is non-empty and mentions a check-ish token (`serve` or `ok`/`fail`/`check`/`version`) — table-ish progress, not silent.

## Side Effects

- None beyond seeds.

## Errors

- Non-nil RunCLI error.

## Exit Code

- Non-nil.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RunErr == "" {
		t.Fatal("expected RunCLI error when doctor critical check fails")
	}
	out := strings.ToLower(resp.Stdout)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected table-ish doctor output on stdout even on failure")
	}
	// At least one of these tokens should appear in a doctor table.
	found := false
	for _, needle := range []string{"serve", "check", "fail", "ok", "version", "profile", "root"} {
		if strings.Contains(out, needle) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stdout lacks doctor check tokens; got:\n%s", resp.Stdout)
	}
}
```
