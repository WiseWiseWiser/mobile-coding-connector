## Expected

1. `DropdownLine` is exactly `Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00, left 26d`.

## Errors

- Old label prefix, em dash, raw codex reset string, or wrong relative unit.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00, left 26d"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```