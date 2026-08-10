## Expected

1. No error from Run.
2. Returned `Published.ID` is non-empty.
3. Returned `Published.TS` is non-empty.
4. Source/type match the request fixture.

## Errors

- Empty id or ts after Publish.
- Run error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}
	if resp.Published.ID == "" {
		t.Fatal("Published.ID empty; want hub-assigned id")
	}
	if resp.Published.TS == "" {
		t.Fatal("Published.TS empty; want hub-assigned ts")
	}
	if resp.Published.Source != req.Event.Source {
		t.Fatalf("Source = %q, want %q", resp.Published.Source, req.Event.Source)
	}
	if resp.Published.Type != req.Event.Type {
		t.Fatalf("Type = %q, want %q", resp.Published.Type, req.Event.Type)
	}
}
```
