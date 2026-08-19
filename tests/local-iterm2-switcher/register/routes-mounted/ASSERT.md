## Expected

1. Inventory, focus, and notes routes are not 404.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.InventoryMounted {
		t.Fatal("inventory not mounted")
	}
	if !resp.FocusMounted {
		t.Fatal("focus not mounted")
	}
	if !resp.NotesMounted {
		t.Fatal("notes not mounted")
	}
	if !resp.StreamMounted {
		t.Fatal("inventory stream not mounted")
	}
}
```
