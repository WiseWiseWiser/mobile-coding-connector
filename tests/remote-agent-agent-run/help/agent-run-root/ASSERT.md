## Expected

1. Exit code 0.
2. Stdout is help for `agent-run` (or `remote-agent agent-run`) and mentions
   the `sessions` subcommand.
3. Stdout is non-empty.

## Errors

- Exit non-zero.
- Help omits `sessions`.

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
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty help stdout")
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "agent-run") && !strings.Contains(lower, "usage") {
		t.Fatalf("expected agent-run/usage in help; stdout:\n%s", out)
	}
	if !strings.Contains(lower, "sessions") {
		t.Fatalf("agent-run help must mention sessions; stdout:\n%s", out)
	}
}
```
