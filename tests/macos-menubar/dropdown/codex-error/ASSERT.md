## Expected

1. `DropdownLine` is exactly `Codex: Error: fork/exec /Users/xhd2015/go/bin/codex-show-status: no such file or directory`.

## Errors

- Truncated or placeholder error in dropdown.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Codex: Error: fork/exec /Users/xhd2015/go/bin/codex-show-status: no such file or directory"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```