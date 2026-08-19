```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FilteredCount != 1 {
		t.Fatalf("filtered=%d want 1 on Desktop 2", resp.FilteredCount)
	}
}
```
