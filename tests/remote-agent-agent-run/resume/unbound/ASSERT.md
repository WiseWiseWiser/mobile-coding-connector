## Expected

1. Non-zero exit.
2. Error mentions bind / runner_session / unbound / cannot resume.

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
		t.Fatalf("unbound resume must fail; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + "\n" + resp.Stderr)
	if !strings.Contains(msg, "bind") && !strings.Contains(msg, "runner_session") &&
		!strings.Contains(msg, "unbound") && !strings.Contains(msg, "cannot resume") &&
		!strings.Contains(msg, "no runner") {
		t.Fatalf("expected unbound/bind error; combined:\n%s", resp.Combined)
	}
}
```
