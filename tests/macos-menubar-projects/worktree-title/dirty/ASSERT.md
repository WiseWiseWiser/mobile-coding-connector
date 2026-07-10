## Expected

1. `Title` is exactly `feat-login ○ Dirty`.

## Errors

- Clean presentation for a dirty worktree.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "feat-login ○ Dirty"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
