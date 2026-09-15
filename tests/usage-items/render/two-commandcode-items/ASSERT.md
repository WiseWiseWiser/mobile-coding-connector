## Expected

1. Both Command Code items report status `ready` with cycle-credit percent titles.
2. Each dropdown line is the label, a colon, and the cycle summary.
3. The pinned item is the one the menu bar shows (default `cc-v1`, rotate off).

## Errors

- Sharing one rendered string between items, or omitting the label prefix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	views := viewsByID(resp.Items)
	wantBody := "5% used, $66.19 left, 1,870 requests, 5h 15%, Weekly 11%, renews in 26d"
	for _, id := range []string{"cc-v1", "cc-v2"} {
		view, ok := views[id]
		if !ok {
			t.Fatalf("item %s missing from %v", id, resp.Items)
		}
		if view.Status != "ready" {
			t.Fatalf("%s status = %q", id, view.Status)
		}
		if view.Title != view.Label+" 5%" {
			t.Fatalf("%s title = %q", id, view.Title)
		}
		if view.Dropdown != view.Label+": "+wantBody {
			t.Fatalf("%s dropdown = %q", id, view.Dropdown)
		}
	}
	if resp.Default != "cc-v1" || resp.Rotate {
		t.Fatalf("selection = %q/rotate=%v", resp.Default, resp.Rotate)
	}
}
```
