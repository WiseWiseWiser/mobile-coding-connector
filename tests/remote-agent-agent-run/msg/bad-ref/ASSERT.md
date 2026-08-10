## Expected

1. Non-zero exit.
2. Error mentions session / message / format / require / `/`.

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
		t.Fatalf("bad ref must fail")
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "session") && !strings.Contains(msg, "message") &&
		!strings.Contains(msg, "require") && !strings.Contains(msg, "/") &&
		!strings.Contains(msg, "format") && !strings.Contains(msg, "invalid") {
		t.Fatalf("expected bad-ref error; combined:\n%s", resp.Combined)
	}
}
```
