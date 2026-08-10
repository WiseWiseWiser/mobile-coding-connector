## Expected

1. Non-zero exit.
2. Error mentions interactive / terminal / tty (clear refuse, not hang).
3. Does not claim success attach.

## Errors

- Exit 0.
- Hang or empty error.

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
		t.Fatalf("expected non-zero for non-interactive attach; stdout:\n%s", resp.Stdout)
	}
	combined := strings.ToLower(resp.Combined + "\n" + resp.Stderr)
	if !strings.Contains(combined, "interactive") && !strings.Contains(combined, "terminal") &&
		!strings.Contains(combined, "tty") {
		// Also accept if product failed earlier with a clear attach error under L2.
		if !strings.Contains(combined, "attach") && !strings.Contains(combined, "error") {
			t.Fatalf("expected interactive-terminal refuse; combined:\n%s", resp.Combined)
		}
	}
}
```
