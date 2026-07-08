## Expected

1. `Title` is exactly `… ⚠ Error`.

## Errors

- Shows running/stopped label instead of error glyph.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "… ⚠ Error" {
		t.Fatalf("title = %q, want %q", resp.Title, "… ⚠ Error")
	}
}
```