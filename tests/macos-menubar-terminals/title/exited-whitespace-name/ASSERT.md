## Expected

1. `Title` is exactly `sess-1 [EXITED]` (whitespace name discarded; exited suffix applied).

## Errors

- Treating whitespace as a visible name, or omitting the exited suffix.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "sess-1 [EXITED]"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
