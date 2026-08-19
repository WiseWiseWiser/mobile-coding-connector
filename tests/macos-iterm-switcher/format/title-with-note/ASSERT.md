```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "grok review  ·  fix auth on staging" {
		t.Fatalf("title=%q", resp.Title)
	}
}
```
