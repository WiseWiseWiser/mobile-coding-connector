## Expected

1. Items are `grok` then `codex`, both enabled, labeled `Grok` and `Codex`.
2. `Default` is `grok` and `Rotate` is true.
3. The seed was persisted to `resp.Registry`.

## Errors

- Seeding an empty registry, dropping items, or defaulting to rotate false.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.Items))
	}
	if resp.Items[0].ID != "grok" || resp.Items[1].ID != "codex" {
		t.Fatalf("ids = %q, %q", resp.Items[0].ID, resp.Items[1].ID)
	}
	for _, view := range resp.Items {
		if !view.Enabled {
			t.Fatalf("item %s must start enabled", view.ID)
		}
	}
	if resp.Items[0].Label != "Grok" || resp.Items[1].Label != "Codex" {
		t.Fatalf("labels = %q, %q", resp.Items[0].Label, resp.Items[1].Label)
	}
	if resp.Default != "grok" || !resp.Rotate {
		t.Fatalf("selection = %q/rotate=%v", resp.Default, resp.Rotate)
	}
	reg := readRegistry(t, resp.Registry)
	if len(reg.Items) != 2 || reg.Version != 1 {
		t.Fatalf("persisted registry = %+v", reg)
	}
}
```
