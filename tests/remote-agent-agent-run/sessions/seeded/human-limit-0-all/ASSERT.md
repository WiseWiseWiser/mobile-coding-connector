## Expected

1. Exit code 0.
2. All 12 seeded session ids appear (`sess-00` … `sess-11`).
3. No “showing N of M” truncation line (limit 0 = all).

## Errors

- Missing any seeded id.
- Non-zero exit.

## Exit Code

0.

```go
import (
	"fmt"
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
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("sess-%02d", i)
		if !strings.Contains(out, id) {
			t.Fatalf("--limit 0 must list all sessions; missing %s; stdout:\n%s", id, out)
		}
	}
	lower := strings.ToLower(out)
	if strings.Contains(lower, "showing") && (strings.Contains(out, " of ") || strings.Contains(out, "of 12")) {
		t.Fatalf("--limit 0 should not print truncation showing-note; stdout:\n%s", out)
	}
}
```
