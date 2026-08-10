## Expected

1. Non-zero exit.
2. Error mentions unreachable / terminal / tty / unavailable.

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
		t.Fatalf("unreachable send must fail")
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "unreachable") && !strings.Contains(msg, "terminal") &&
		!strings.Contains(msg, "tty") && !strings.Contains(msg, "unavailable") {
		t.Fatalf("expected unreachable error; combined:\n%s", resp.Combined)
	}
}
```
