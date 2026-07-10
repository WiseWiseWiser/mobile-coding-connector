## Expected

1. `Leading` is exactly `feat-login`.
2. `Trailing` is exactly `○ Dirty`.
3. Legacy `Title` is exactly `feat-login  ○ Dirty`.

## Errors

- Clean presentation for a dirty worktree.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Leading != "feat-login" {
		t.Fatalf("Leading = %q, want %q", resp.Leading, "feat-login")
	}
	if resp.Trailing != "○ Dirty" {
		t.Fatalf("Trailing = %q, want %q", resp.Trailing, "○ Dirty")
	}
	wantTitle := "feat-login  ○ Dirty"
	if resp.Title != wantTitle {
		t.Fatalf("Title = %q, want %q", resp.Title, wantTitle)
	}
}
```
