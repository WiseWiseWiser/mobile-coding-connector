## Expected

1. `Title` is exactly `demo` (name wins over id).

## Errors

- Using id when name is non-empty, or appending cwd/status to title.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "demo" {
		t.Fatalf("title = %q, want %q", resp.Title, "demo")
	}
}
```
