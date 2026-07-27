## Expected

1. Label exactly `Local Dashboard`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.Label != "Local Dashboard" {
		t.Fatalf("label=%q", resp.Label)
	}
}
```
