## Expected

1. Non-zero exit.
2. Error mentions session / message / require / usage / argument.

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
		t.Fatalf("expected non-zero without args")
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "session") && !strings.Contains(msg, "message") &&
		!strings.Contains(msg, "usage") && !strings.Contains(msg, "require") &&
		!strings.Contains(msg, "argument") {
		t.Fatalf("expected clear args error; combined:\n%s", resp.Combined)
	}
}
```
