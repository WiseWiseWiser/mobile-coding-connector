## Expected

1. `DropdownLine` is exactly `Grok: Error: timeout waiting`.

## Errors

- Placeholder ellipsis or missing error text in dropdown.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: Error: timeout waiting"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```