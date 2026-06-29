## Expected

1. Exit 0.
2. Stdout is `Test User` (trimmed).

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s", resp.ExitCode, resp.Stdout)
	}
	if strings.TrimSpace(resp.Stdout) != "Test User" {
		t.Fatalf("got user.name %q", resp.Stdout)
	}
}
```