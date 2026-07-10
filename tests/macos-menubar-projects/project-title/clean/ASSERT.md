## Expected

1. `Leading` is exactly `demo` (basename only).
2. `Trailing` is exactly `● main` (filled circle + branch).
3. Legacy `Title` is exactly `demo  ● main` (`Leading + "  " + Trailing`).

## Errors

- Wrong glyph (○ instead of ●), missing branch, or trailing folded into leading.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Leading != "demo" {
		t.Fatalf("Leading = %q, want %q", resp.Leading, "demo")
	}
	if resp.Trailing != "● main" {
		t.Fatalf("Trailing = %q, want %q", resp.Trailing, "● main")
	}
	wantTitle := "demo  ● main"
	if resp.Title != wantTitle {
		t.Fatalf("Title = %q, want %q (Leading + \"  \" + Trailing)", resp.Title, wantTitle)
	}
}
```
