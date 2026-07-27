## Expected

1. Exit 0 (idle shutdown is success).
2. Stdout mentions a public URL (`https://`) and provider name (quick / cloudflare_quick).

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
	out := resp.Stdout + resp.Stderr
	if !strings.Contains(out, "https://") {
		t.Fatalf("expected public URL; combined:\n%s", resp.Combined)
	}
	low := strings.ToLower(out)
	if !strings.Contains(low, "quick") && !strings.Contains(low, "cloudflare") {
		t.Fatalf("expected provider in output; combined:\n%s", resp.Combined)
	}
}
```
