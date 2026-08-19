## Expected

1. View contains title `Terminals`.
2. Sidebar tokens: `All` and `Desktop 1`.
3. Session primary `grok review`.
4. Status token `cached`.
5. Split box drawing (`┌` or `│`) — not a chip-row-only layout.

```go
import (
	"regexp"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	view := resp.ViewText
	if view == "" {
		t.Fatal("View empty (split TUI not implemented)")
	}
	for _, tok := range []string{`Terminals`, `All`, `Desktop 1`, `grok review`, `cached`} {
		if !regexp.MustCompile(regexp.QuoteMeta(tok)).MatchString(view) {
			t.Fatalf("warm-split missing %q; view=%q", tok, view)
		}
	}
	if !regexp.MustCompile(`┌|│`).MatchString(view) {
		t.Fatalf("warm-split must paint split box (┌ or │); view=%q", view)
	}
}
```
