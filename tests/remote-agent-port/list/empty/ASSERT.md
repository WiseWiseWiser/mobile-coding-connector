## Expected

1. Exit 0.
2. Stdout indicates no listening ports (empty message; not a table with rows).
3. Stdout ends with newline.

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
	out := strings.ToLower(resp.Stdout)
	if !strings.Contains(out, "no") && !strings.Contains(out, "empty") && strings.TrimSpace(resp.Stdout) != "" {
		// Accept common empty messages; reject silent non-empty garbage tables
		if strings.Contains(resp.Stdout, "3000") {
			t.Fatalf("empty list should not show ports; stdout:\n%s", resp.Stdout)
		}
	}
	if strings.Contains(resp.Stdout, "3000") {
		t.Fatalf("empty list leaked a port; stdout:\n%s", resp.Stdout)
	}
	if strings.TrimSpace(resp.Stdout) == "" {
		t.Fatalf("empty list should print a clear empty message, not silent stdout")
	}
}
```
