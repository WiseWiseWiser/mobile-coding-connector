## Expected

1. `Title` is exactly `demo ⚠ Error` (full name + error presentation).
2. Error state wins over branch/clean fields.

## Errors

- Shows branch/clean instead of error glyph.
- Hides project name.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "demo ⚠ Error"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
