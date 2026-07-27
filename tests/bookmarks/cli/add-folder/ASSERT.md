## Expected

1. Exit 0.
2. fld_work type folder.

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
	n := FindNode(resp.Doc, "fld_work")
	if n == nil {
		n = FindNodeByName(resp.Doc, "Work")
	}
	if n == nil || n.Type != "folder" {
		t.Fatalf("folder missing: %+v doc=%+v", n, resp.Doc)
	}
}
```
