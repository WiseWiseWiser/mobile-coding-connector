```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FilteredCount != 2 {
		t.Fatalf("filtered=%d want 2 starred", resp.FilteredCount)
	}
}
```
