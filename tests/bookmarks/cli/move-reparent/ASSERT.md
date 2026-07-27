## Expected

1. Exit 0.
2. bm_cm under fld_c.

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
	fld := FindNode(resp.Doc, "fld_c")
	if fld == nil {
		t.Fatal("fld_c missing")
	}
	for _, c := range fld.Children {
		if c.ID == "bm_cm" {
			return
		}
	}
	t.Fatalf("bm_cm not under fld_c; doc=%+v", resp.Doc)
}
```
