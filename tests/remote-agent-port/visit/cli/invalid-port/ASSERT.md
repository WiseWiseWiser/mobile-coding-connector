## Expected

1. Non-zero exit.
2. Stderr contains `Error:` and mentions port.

## Exit Code

non-zero.

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
		t.Fatalf("expected non-zero for invalid port; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stderr, "Error:") && !strings.Contains(resp.Combined, "Error:") {
		t.Fatalf("expected Error: on stderr; combined:\n%s", resp.Combined)
	}
}
```
