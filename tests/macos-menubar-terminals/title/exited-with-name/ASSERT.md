## Expected

1. `Title` is exactly `demo [EXITED]` (name base + exact exited suffix).

## Errors

- Missing suffix, wrong case (`[Exited]`), missing leading space, or using id when name is set.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "demo [EXITED]"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
