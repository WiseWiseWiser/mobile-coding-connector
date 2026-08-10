## Expected

1. Non-zero exit.
2. Error mentions not found / unknown / missing / session.

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
		t.Fatalf("expected error for missing session; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + "\n" + resp.Stderr)
	if !strings.Contains(msg, "not found") && !strings.Contains(msg, "unknown") &&
		!strings.Contains(msg, "missing") && !strings.Contains(msg, "no such") &&
		!strings.Contains(msg, "session") {
		t.Fatalf("expected clear missing-session error; combined:\n%s", resp.Combined)
	}
}
```
