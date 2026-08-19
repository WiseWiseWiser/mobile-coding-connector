## Expected

1. `localiterm2.Register` is in server.go.
2. Inventory/focus/notes are not in the auth skip list.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.RegisteredInServer {
		t.Fatal("server.go must call localiterm2.Register")
	}
	if resp.InAuthSkipList {
		t.Fatal("switcher paths must not be auth skip-listed")
	}
}
```
