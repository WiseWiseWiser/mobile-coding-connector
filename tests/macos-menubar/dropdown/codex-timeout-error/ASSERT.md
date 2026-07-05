## Expected

1. `DropdownLine` is exactly `Codex: Error: timeout waiting for status output`.

## Errors

- Truncated or placeholder error in dropdown.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Codex: Error: timeout waiting for status output"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```