## Expected

1. `DropdownLine` is exactly `Codex: Monthly Usage: 58% — 6,519/11,250 (Reset 08:00 on 1 Aug)`.

## Errors

- Wrong em dash, credits fraction, or reset formatting.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Codex: Monthly Usage: 58% — 6,519/11,250 (Reset 08:00 on 1 Aug)"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```