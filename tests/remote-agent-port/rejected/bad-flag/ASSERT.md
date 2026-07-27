## Expected

1. Non-zero exit.
2. Error: about unknown/invalid flag.

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
		t.Fatal("expected non-zero for bad flag")
	}
	if !strings.Contains(resp.Combined, "Error:") {
		t.Fatalf("expected Error:; combined:\n%s", resp.Combined)
	}
}
```
