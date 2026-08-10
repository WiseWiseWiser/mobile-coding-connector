## Expected

1. Exit 0.
2. Mentions `resume` and session id.
3. Documents `--open` (resume then attach).

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
	if !strings.Contains(out, "resume") {
		t.Fatalf("help must mention resume; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(out, "session") {
		t.Fatalf("resume help must document session-id; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(out, "--open") && !strings.Contains(out, "open") {
		t.Fatalf("resume help should document --open; stdout:\n%s", resp.Stdout)
	}
}
```
