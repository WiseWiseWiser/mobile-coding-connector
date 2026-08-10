## Expected

1. Exit 0.
2. Soft: MsgCancelCalled when inject wired.

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
		t.Fatalf("msg cancel failed: %s", resp.Combined)
	}
	msg := strings.ToLower(resp.Combined)
	if strings.Contains(msg, "unknown") && strings.Contains(msg, "subcommand") {
		t.Fatalf("msg cancel not wired: %s", resp.Combined)
	}
}
```
