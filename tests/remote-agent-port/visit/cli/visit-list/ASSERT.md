## Expected

1. Exit 0.
2. Stdout indicates no active visits (clear empty message) or empty JSON if --json not set: human empty text.

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
	if strings.TrimSpace(resp.Stdout) == "" {
		t.Fatal("visit list empty should print a clear empty message")
	}
}
```
