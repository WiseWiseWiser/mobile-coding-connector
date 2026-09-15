## Expected

1. Items keep registry order: grok, codex, cc-v1.
2. The disabled codex item is never fetched, so it stays `loading`.
3. The enabled items both render ready text.

## Errors

- Fetching disabled items, or reordering the registry while rendering.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(resp.Items))
	}
	if resp.Items[0].ID != "grok" || resp.Items[1].ID != "codex" || resp.Items[2].ID != "cc-v1" {
		t.Fatalf("order = %q, %q, %q", resp.Items[0].ID, resp.Items[1].ID, resp.Items[2].ID)
	}
	if resp.Items[1].Status != "loading" || resp.Items[1].Title != "Codex ..." {
		t.Fatalf("disabled item = %+v", resp.Items[1])
	}
	if resp.Items[0].Status != "ready" || resp.Items[2].Status != "ready" {
		t.Fatalf("enabled items = %+v", resp.Items)
	}
	if !resp.Rotate {
		t.Fatal("rotation must stay on")
	}
}
```
