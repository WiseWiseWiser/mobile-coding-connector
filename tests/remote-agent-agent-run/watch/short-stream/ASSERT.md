## Expected

1. Exit 0 (stream completed under L2 inject).
2. Stdout contains inject lines `WATCH_A` and `WATCH_B`.

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
		t.Fatalf("watch short-stream failed: %s", resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "WATCH_A") || !strings.Contains(resp.Stdout, "WATCH_B") {
		t.Fatalf("expected watch lines; stdout:\n%s", resp.Stdout)
	}
}
```
