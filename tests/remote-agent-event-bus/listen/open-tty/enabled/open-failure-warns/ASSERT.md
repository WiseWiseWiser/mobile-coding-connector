## Expected

1. Exit code 0 (open failure is best-effort; MaxEvents path succeeds).
2. OpenTTYSession was attempted once with fixtureSessionIDA.
3. Stderr/combined contains `warning:` about open failure.
4. Event still printed.

## Errors

- Open error fails the whole listen (non-zero exit).
- Silent open failure with no warning.
- Hook never invoked.

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
		t.Fatalf("open failure must not fail listen; exit %d; combined:\n%s",
			resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, fixtureTypeTTY) {
		t.Fatalf("expected printed event; stdout:\n%s", resp.Stdout)
	}
	if len(resp.OpenTTYSessionIDs) != 1 || resp.OpenTTYSessionIDs[0] != fixtureSessionIDA {
		t.Fatalf("expected one open attempt for %q; got %v", fixtureSessionIDA, resp.OpenTTYSessionIDs)
	}
	if !strings.Contains(resp.Stderr, "warning:") && !strings.Contains(resp.Combined, "warning:") {
		t.Fatalf("expected warning: on open failure; stderr:\n%s\nstdout:\n%s",
			resp.Stderr, resp.Stdout)
	}
}
```
