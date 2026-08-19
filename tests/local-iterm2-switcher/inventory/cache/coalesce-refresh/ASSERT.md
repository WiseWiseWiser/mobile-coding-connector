```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CaptureCalls != 2 {
		t.Fatalf("CaptureCalls=%d want 2 (warm + one coalesced refresh, not 1+3)", resp.CaptureCalls)
	}
}
```
