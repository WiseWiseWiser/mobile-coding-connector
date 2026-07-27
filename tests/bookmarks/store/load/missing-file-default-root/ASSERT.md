## Expected

1. `Run` harness error is nil.
2. `ErrMsg` empty.
3. `Doc.Version == 1`.
4. One root: `id=root`, `type=folder`, `name=Bookmarks`, `len(children)==0`.

## Side Effects

- Manager may create default file on first mutation; load alone may not require write.

## Errors

- Non-default root or version ≠ 1.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ErrMsg != "" {
		t.Fatalf("unexpected ErrMsg: %s", resp.ErrMsg)
	}
	if resp.Doc == nil {
		t.Fatal("Doc is nil")
	}
	if !defaultRootOK(resp.Doc) {
		t.Fatalf("default root not OK: %+v", resp.Doc)
	}
	if len(resp.Doc.Roots[0].Children) != 0 {
		t.Fatalf("want empty children, got %d", len(resp.Doc.Roots[0].Children))
	}
}
```
