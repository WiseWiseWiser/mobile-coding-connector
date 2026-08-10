## Expected

1. Exit code 0 after MaxEvents=2.
2. Stderr or stdout contains `warning:` about disconnect/reconnect (prefer stderr).
3. Both event types/ids appear in output (human or recoverable stream).
4. At least one `connected` (initial; second connect after retry optional but allowed).

## Errors

- Silent disconnect with no warning.
- Hang without second event.
- Exit non-zero after successful recovery.

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
	combined := resp.Combined
	if !strings.Contains(combined, "warning:") {
		t.Fatalf("expected warning: on disconnect/reconnect; combined:\n%s", combined)
	}
	// Prefer warning on stderr (product style for warnings).
	if !strings.Contains(resp.Stderr, "warning:") && !strings.Contains(resp.Stdout, "warning:") {
		t.Fatalf("warning: missing; stderr:\n%s\nstdout:\n%s", resp.Stderr, resp.Stdout)
	}
	out := resp.Stdout
	typeCount := strings.Count(out, fixtureTypeSeatalk)
	has1 := strings.Contains(out, fixtureEventID1) || strings.Contains(out, "hello-bus")
	has2 := strings.Contains(out, fixtureEventID2) || strings.Contains(out, "second")
	if typeCount < 2 && !(has1 && has2) {
		t.Fatalf("expected two printed events after retry (typeCount=%d has1=%v has2=%v); stdout:\n%s",
			typeCount, has1, has2, out)
	}
}
```
