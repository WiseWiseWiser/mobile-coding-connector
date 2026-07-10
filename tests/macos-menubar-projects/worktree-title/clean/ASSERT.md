## Expected

1. `Leading` is exactly `feat-login`.
2. `Trailing` is exactly `● Clean`.
3. Legacy `Title` is exactly `feat-login  ● Clean`.

## Errors

- Missing basename or wrong clean marker.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Leading != "feat-login" {
		t.Fatalf("Leading = %q, want %q", resp.Leading, "feat-login")
	}
	if resp.Trailing != "● Clean" {
		t.Fatalf("Trailing = %q, want %q", resp.Trailing, "● Clean")
	}
	wantTitle := "feat-login  ● Clean"
	if resp.Title != wantTitle {
		t.Fatalf("Title = %q, want %q", resp.Title, wantTitle)
	}
}
```
