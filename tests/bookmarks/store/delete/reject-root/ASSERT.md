## Expected

1. ErrMsg non-empty (product error).
2. `defaultRootOK(Doc)` still true.

## Errors

- Silent success deleting root.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ErrMsg == "" {
		t.Fatal("expected error deleting root")
	}
	if !defaultRootOK(resp.Doc) {
		t.Fatalf("root must remain: %+v", resp.Doc)
	}
}
```
