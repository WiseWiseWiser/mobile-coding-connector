## Expected

Actual process exit is explicit both live and on reattachment.

```go
import (
    "testing"
    "github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
    if err != nil { t.Fatal(err) }
    if resp.Status != "exited" { t.Fatalf("expected confirmed process exit, got %+v", resp) }
}
```
