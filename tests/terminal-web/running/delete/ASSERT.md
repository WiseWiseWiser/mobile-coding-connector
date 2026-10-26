## Expected

Explicit tab deletion removes the session.

```go
import (
    "testing"
    "github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
    if err != nil { t.Fatal(err) }
    if resp.Status != "running" || !resp.SameSession || !resp.Replay || !resp.Input || !resp.Pong {
        t.Fatalf("same live PTY must replay, accept input and answer heartbeat: %+v", resp)
    }
    if resp.SessionCount != 0 { t.Fatalf("unexpected session count: %+v", resp) }
}
```
