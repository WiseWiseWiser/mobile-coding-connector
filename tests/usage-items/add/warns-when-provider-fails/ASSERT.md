## Expected

1. The add succeeds and returns exactly one warning naming the item and the error.
2. The registered item reports status `error`.
3. The menu bar text is `CC dead err` / `CC dead: Error: Session expired`.

## Errors

- Failing the add, or hiding the provider error from the warning.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0] != "cc-dead fetch failed: Session expired" {
		t.Fatalf("warnings = %v", resp.Warnings)
	}
	if resp.Item == nil {
		t.Fatal("the item must still be registered")
	}
	if resp.Item.Status != "error" || resp.Item.Error != "Session expired" {
		t.Fatalf("item = %+v", resp.Item)
	}
	if resp.Item.Title != "CC dead err" {
		t.Fatalf("title = %q, want %q", resp.Item.Title, "CC dead err")
	}
	if resp.Item.Dropdown != "CC dead: Error: Session expired" {
		t.Fatalf("dropdown = %q", resp.Item.Dropdown)
	}
}
```
