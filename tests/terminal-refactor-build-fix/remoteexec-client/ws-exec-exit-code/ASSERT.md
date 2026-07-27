## Expected

- Client stdout contains `hello from remote`.
- `RunInteractive` returns exit code 42 without error.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("RunInteractive error: %v (stdout=%q)", err, resp.WSStdout)
	}
	if resp.WSExitCode != 42 {
		t.Fatalf("exit code: got %d, want 42", resp.WSExitCode)
	}
	if !strings.Contains(resp.WSStdout, "hello from remote") {
		t.Fatalf("stdout %q missing payload", resp.WSStdout)
	}
}
```