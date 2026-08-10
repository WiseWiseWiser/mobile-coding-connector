## Expected

1. Exit 0.
2. Stdout contains session id `sess-run-1` and preferably terminal id `term-run-1`.
3. Soft: RunCalled && RunDetach when inject wired.

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
		t.Fatalf("run --detach failed: %s", resp.Combined)
	}
	out := resp.Stdout + "\n" + resp.Combined
	if !strings.Contains(out, "sess-run-1") {
		t.Fatalf("expected session id on output; got:\n%s", out)
	}
	// terminal id preferred but soft if product only prints session
	if resp.RunCalled && !resp.RunDetach {
		t.Fatalf("expected RunDetach=true when inject observed")
	}
	msg := strings.ToLower(out)
	if strings.Contains(msg, "unknown") && strings.Contains(msg, "subcommand") {
		t.Fatalf("run not wired: %s", out)
	}
}
```
