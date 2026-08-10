## Expected

1. Exit 0.
2. Stdout contains message id `msg_1` (or inject id).
3. Soft: SendCalled when inject wired.

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
		t.Fatalf("send success failed: %s", resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "msg_1") && !strings.Contains(resp.Stdout, "msg_") {
		t.Fatalf("expected message id on stdout; got:\n%s", resp.Stdout)
	}
}
```
