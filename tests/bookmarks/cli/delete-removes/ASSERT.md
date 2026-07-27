## Expected

1. Exit 0.
2. bm_del absent.

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
	if FindNode(resp.Doc, "bm_del") != nil {
		t.Fatal("bm_del still present")
	}
}
```
