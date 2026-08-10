## Expected

1. Non-zero exit.
2. Error indicates not available via remote-agent / unsupported / local-only /
   new-terminal rejected (clear, not hang).

## Exit Code

non-zero

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
		t.Fatalf("--new-terminal must be rejected; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + resp.Stderr)
	if !strings.Contains(msg, "new-terminal") && !strings.Contains(msg, "new terminal") &&
		!strings.Contains(msg, "not available") && !strings.Contains(msg, "unsupported") &&
		!strings.Contains(msg, "local-only") && !strings.Contains(msg, "remote") &&
		!strings.Contains(msg, "iterm") {
		t.Fatalf("expected clear --new-terminal rejection; combined:\n%s", resp.Combined)
	}
}
```
