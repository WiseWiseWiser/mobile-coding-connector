## Expected

A missing session produces a lifecycle error, not a new shell.

```go
import (
    "testing"
    "github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
    if err != nil { t.Fatal(err) }
    if resp.Status != "missing" || resp.SessionCount != 0 { t.Fatalf("missing session must not spawn a replacement: %+v", resp) }
}
```
