## Expected

1. Non-zero exit.
2. Error mentions prompt / session / usage / require / argument / flag.
3. Not a silent hang (Run returns).

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
		t.Fatalf("bare run without inputs should fail validation; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "prompt") && !strings.Contains(msg, "session") &&
		!strings.Contains(msg, "usage") && !strings.Contains(msg, "require") &&
		!strings.Contains(msg, "argument") && !strings.Contains(msg, "flag") &&
		!strings.Contains(msg, "run") {
		t.Fatalf("expected validation error; combined:\n%s", resp.Combined)
	}
}
```
