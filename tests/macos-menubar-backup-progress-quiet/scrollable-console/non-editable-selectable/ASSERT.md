## Expected

1. `NonEditableSelectable` is true (`isEditable = false` and `isSelectable = true`).

## Side Effects

- None (read-only source inspection).

## Errors

- Editable console or non-selectable text.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.NonEditableSelectable {
		t.Fatalf("expected isEditable=false and isSelectable=true (source: %s)", resp.ProgressWindowSource)
	}
}
```
