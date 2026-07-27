## Expected

1. Exit 0.
2. Stdout includes persistent forward public URL or port 4000.
3. Stdout still includes local listener 3000.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(out, "3000") {
		t.Fatalf("missing local listener; stdout:\n%s", out)
	}
	if !strings.Contains(out, "4000") && !strings.Contains(out, seedForwardURL) {
		t.Fatalf("missing persistent forward; stdout:\n%s", out)
	}
}
```
