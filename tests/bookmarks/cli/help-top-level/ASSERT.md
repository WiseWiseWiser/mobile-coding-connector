## Expected

1. Combined output mentions `list`, `add`, `delete`, and `open` (case-insensitive ok).
2. Exit code may be 0 or help-style non-zero depending on less-gen; do not require 0 only if help text present.

## Errors

- Unknown command / no help text.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	out := strings.ToLower(resp.Combined)
	if out == "" {
		// help may only return error without capture if command missing
		out = strings.ToLower(resp.ErrMsg + resp.Stdout + resp.Stderr)
	}
	for _, want := range []string{"list", "add", "delete", "open"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q; out=%q err=%q", want, resp.Combined, resp.ErrMsg)
		}
	}
}
```
