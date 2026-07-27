## Expected

1. Exit 0.
2. bm_set name RenamedCLI.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q", resp.ExitCode, resp.Combined)
	}
	n := FindNode(resp.Doc, "bm_set")
	if n == nil || n.Name != "RenamedCLI" {
		t.Fatalf("rename failed: %+v", n)
	}
}
```
