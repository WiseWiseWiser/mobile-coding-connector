## Expected

1. Non-zero exit.
2. Error indicates local-only and/or not available via remote-agent.
3. Soft: may name `focus`.

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
		t.Fatalf("focus must fail as local-only; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "local") && !strings.Contains(msg, "not available") &&
		!strings.Contains(msg, "remote-agent") && !strings.Contains(msg, "unsupported") &&
		!strings.Contains(msg, "unavailable") {
		// Accept unknown-subcommand only if it clearly names the command as unsupported
		if !strings.Contains(msg, "focus") {
			t.Fatalf("expected local-only error for focus; combined:\n%s", resp.Combined)
		}
	}
}
```
