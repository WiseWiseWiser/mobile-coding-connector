## Expected

1. Exit 0.
2. Stdout contains `pending` (or inject status token).

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
		t.Fatalf("msg status failed: %s", resp.Combined)
	}
	if !strings.Contains(strings.ToLower(resp.Stdout), "pending") {
		t.Fatalf("expected pending status; stdout:\n%s", resp.Stdout)
	}
}
```
