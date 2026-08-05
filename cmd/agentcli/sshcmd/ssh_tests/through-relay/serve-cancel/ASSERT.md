## Expected

1. Harness `err` is nil.
2. Happy path same as through-relay remote-command: ServeErr empty, RunnerErr empty, Stdout contains needle.
3. After cancel: `SessionAfterStop` is nil **or** `!Alive`.
4. `PortClosedAfterStop` is true.

## Side Effects

- Session cleared under store root; relay listen closed.

## Errors

None expected on clean cancel after success.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.ServeErr != "" {
		t.Fatalf("ServeService error: %s", resp.ServeErr)
	}
	if resp.RunnerErr != "" {
		t.Fatalf("command before cancel failed: %s", resp.RunnerErr)
	}
	needle := req.EchoNeedle
	if needle == "" {
		needle = "hello"
	}
	if !strings.Contains(resp.Stdout, needle) {
		t.Fatalf("Stdout must contain %q before cancel; got %q", needle, resp.Stdout)
	}
	if resp.SessionAfterStop != nil && resp.SessionAfterStop.Alive {
		t.Fatal("SessionAfterStop still Alive; cancel must Clear or mark not alive")
	}
	if !resp.PortClosedAfterStop {
		t.Fatal("relay listen port must be closed after cancel")
	}
}
```
