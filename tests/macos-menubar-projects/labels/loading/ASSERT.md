## Expected

1. `Label` is exactly `Loading…` (capital L, unicode ellipsis `…`).

## Errors

- ASCII triple-dot `Loading...` instead of `Loading…`.
- Different casing or wording (`loading`, `Please wait`).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	// Unicode ellipsis U+2026 — not three ASCII periods.
	want := "Loading…"
	if resp.Label != want {
		t.Fatalf("Label = %q, want %q", resp.Label, want)
	}
}
```
