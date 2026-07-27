## Expected

1. `DropdownLine` is exactly `Grok: 6%(Weekly), Reset July 9, 16:55, left 3d`.

## Errors

- Old parenthetical format, raw PT suffix, or wrong relative unit.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: 6%(Weekly), Reset July 9, 16:55, left 3d"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
}
```