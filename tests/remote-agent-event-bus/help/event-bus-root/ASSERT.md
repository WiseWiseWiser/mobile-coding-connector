## Expected

1. Exit code 0.
2. Stdout is help for `event-bus` and mentions `listen` subcommand.
3. Stdout contains `Usage`.

## Errors

- Unknown command: event-bus.
- Help omits listen.

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
	out := resp.Stdout
	if !strings.Contains(strings.ToLower(out), "usage") {
		t.Fatalf("expected Usage in event-bus help; stdout:\n%s", out)
	}
	if !strings.Contains(out, "listen") {
		t.Fatalf("event-bus help must mention listen; stdout:\n%s", out)
	}
	if !strings.Contains(out, "event-bus") {
		t.Fatalf("event-bus help should mention event-bus; stdout:\n%s", out)
	}
}
```
