## Expected

1. Exit 0.
2. Mentions kill, session, and `--dry-run`.

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
		t.Fatalf("exit %d; %s", resp.ExitCode, resp.Combined)
	}
	out := strings.ToLower(resp.Stdout)
	if !strings.Contains(out, "kill") {
		t.Fatalf("help must mention kill; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(out, "session") {
		t.Fatalf("kill help must document session-id; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(out, "dry-run") && !strings.Contains(out, "dry run") {
		t.Fatalf("kill help must document --dry-run; stdout:\n%s", resp.Stdout)
	}
}
```
