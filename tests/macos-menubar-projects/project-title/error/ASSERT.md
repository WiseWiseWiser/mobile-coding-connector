## Expected

1. `Leading` is exactly `demo` (name still shown).
2. `Trailing` is exactly `⚠ Error` (error presentation wins over branch/clean).
3. Legacy `Title` is exactly `demo  ⚠ Error`.

## Errors

- Shows branch/clean instead of error glyph.
- Hides project name in Leading.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Leading != "demo" {
		t.Fatalf("Leading = %q, want %q", resp.Leading, "demo")
	}
	if resp.Trailing != "⚠ Error" {
		t.Fatalf("Trailing = %q, want %q", resp.Trailing, "⚠ Error")
	}
	wantTitle := "demo  ⚠ Error"
	if resp.Title != wantTitle {
		t.Fatalf("Title = %q, want %q", resp.Title, wantTitle)
	}
}
```
