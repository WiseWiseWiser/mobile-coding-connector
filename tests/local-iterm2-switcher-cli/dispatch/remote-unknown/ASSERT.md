## Expected

1. Every alias yields non-zero exit with `unknown command: <typed name>`.
2. MultiOK true; MultiChecked == 4.

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
		t.Fatalf("remote must reject each typed name; detail=%q combined=%q", resp.MultiDetail, resp.Combined)
	}
}
```
