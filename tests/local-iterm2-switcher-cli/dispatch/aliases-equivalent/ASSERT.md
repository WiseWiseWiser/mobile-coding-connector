## Expected

1. MultiChecked == 4.
2. MultiOK: each alias exits 0 with Usage `native-terminals` and `--json`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.MultiChecked != 4 {
		t.Fatalf("MultiChecked=%d want 4", resp.MultiChecked)
	}
	if !resp.MultiOK {
		t.Fatalf("aliases not equivalent for list -h; detail=%q combined=%q", resp.MultiDetail, resp.Combined)
	}
}
```
