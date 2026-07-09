## Expected

1. `Title` is exactly `sess-1 [EXITED]` (id base + exact exited suffix).

## Errors

- Suffix without base id, or base without suffix.

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
