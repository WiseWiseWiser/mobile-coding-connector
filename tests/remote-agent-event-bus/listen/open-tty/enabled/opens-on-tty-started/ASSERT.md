## Expected

1. Exit code 0 (MaxEvents stop).
2. Stdout still logs `agent.tty.started` (open does not replace logging).
3. `OpenTTYSession` called exactly once with `fixtureSessionIDA`.

## Errors

- Zero open calls (feature missing).
- Wrong session_id passed to open.
- Non-zero exit after successful receive.

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
		t.Fatalf("expected printed type %q; stdout:\n%s", fixtureTypeTTY, resp.Stdout)
	}
	if len(resp.OpenTTYSessionIDs) != 1 {
		t.Fatalf("expected exactly 1 OpenTTYSession call, got %v; combined:\n%s",
			resp.OpenTTYSessionIDs, resp.Combined)
	}
	if resp.OpenTTYSessionIDs[0] != fixtureSessionIDA {
		t.Fatalf("OpenTTYSession session_id want %q got %q", fixtureSessionIDA, resp.OpenTTYSessionIDs[0])
	}
}
```
