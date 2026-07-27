## Expected

1. `DropdownLine` is exactly `Grok: 61%(Weekly), Reset July 17, 08:55`.
2. Line does not contain `left`.

## Errors

- Appending `, left …` or dropping `Reset …` when time_left is empty.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok: 61%(Weekly), Reset July 17, 08:55"
	if resp.DropdownLine != want {
		t.Fatalf("dropdown = %q, want %q", resp.DropdownLine, want)
	}
	if strings.Contains(resp.DropdownLine, "left") {
		t.Fatalf("dropdown must not contain left suffix: %q", resp.DropdownLine)
	}
}
```
