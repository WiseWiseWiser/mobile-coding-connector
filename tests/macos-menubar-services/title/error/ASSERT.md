## Expected

1. `Title` is exactly `web ⚠ Error` (full service name + error presentation).

## Errors

- Shows running/stopped label instead of error glyph.
- Truncates or hides the service name (e.g. `… ⚠ Error`).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "web ⚠ Error" {
		t.Fatalf("title = %q, want %q", resp.Title, "web ⚠ Error")
	}
}
```
