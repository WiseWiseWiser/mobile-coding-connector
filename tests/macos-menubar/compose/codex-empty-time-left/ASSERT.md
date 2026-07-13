## Expected

1. `DropdownLine` is exactly `Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00`.
2. Line does not contain `left`.

## Errors

- Appending `, left …` when time_left is empty.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Codex: 58%(Monthly) 6,519/11,250, Reset Aug 1, 08:00"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
	if strings.Contains(resp.DropdownLine, "left") {
		t.Fatalf("dropdown must not contain left suffix: %q", resp.DropdownLine)
	}
}
```
