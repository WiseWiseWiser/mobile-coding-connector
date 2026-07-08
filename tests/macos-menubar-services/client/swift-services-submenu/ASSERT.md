## Expected

1. `HasNestedServiceMenu` is `true` — per-service nested `Menu` with action buttons.

## Side Effects

- None (read-only source inspection).

## Errors

- Flat button list without nested submenu per service.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasNestedServiceMenu {
		t.Fatalf("missing nested per-service Menu (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```