## Expected

1. `Title` is exactly `demo ○ main` (name + dirty marker + branch).

## Errors

- Uses clean ● glyph instead of dirty ○.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "demo ○ main"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
