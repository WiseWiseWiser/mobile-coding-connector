## Expected

1. Exit 0.
2. Stdout contains `SANITIZED_SNAPSHOT_LINE`.

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
		t.Fatalf("snapshot failed: %s", resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "SANITIZED_SNAPSHOT_LINE") {
		t.Fatalf("expected snapshot text; stdout:\n%s", resp.Stdout)
	}
}
```
