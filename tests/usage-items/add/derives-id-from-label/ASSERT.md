## Expected

1. The item id is `cc-v2`, the slug of its label.
2. The label is kept as given, not replaced by the provider name.

## Errors

- Storing an empty id, or overwriting an explicit label with the provider name.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Added) != 1 {
		t.Fatalf("added = %d, want 1", len(resp.Added))
	}
	if resp.Added[0].ID != "cc-v2" {
		t.Fatalf("id = %q, want cc-v2", resp.Added[0].ID)
	}
	if resp.Added[0].Label != "CC v2" {
		t.Fatalf("label = %q, want CC v2", resp.Added[0].Label)
	}
}
```
