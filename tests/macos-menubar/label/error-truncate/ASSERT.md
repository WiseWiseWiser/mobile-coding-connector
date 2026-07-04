## Expected

1. `Label` starts with `Grok `.
2. `len([]rune(Label))` is less than or equal to `Response.MaxLabelLen`.
3. `Label` is not the full untruncated error string.
4. Truncated label ends with `…` or `...` (ellipsis marker).

## Errors

- Full long error shown in menu bar label.

```go
import (
	"strings"
	"testing"
	"unicode/utf8"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp.Label, "Grok ") {
		t.Fatalf("label = %q, want Grok prefix", resp.Label)
	}
	if resp.MaxLabelLen <= 0 {
		t.Fatal("MaxLabelLen not set by Run")
	}
	if n := utf8.RuneCountInString(resp.Label); n > resp.MaxLabelLen {
		t.Fatalf("label rune count = %d, want <= %d: %q", n, resp.MaxLabelLen, resp.Label)
	}
	full := "Grok " + req.ErrorMsg
	if resp.Label == full {
		t.Fatal("label was not truncated")
	}
	if !strings.HasSuffix(resp.Label, "…") && !strings.HasSuffix(resp.Label, "...") {
		t.Fatalf("truncated label should end with ellipsis: %q", resp.Label)
	}
}
```