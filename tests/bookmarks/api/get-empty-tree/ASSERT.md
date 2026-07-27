## Expected

1. HTTPStatus 200.
2. Doc default root empty children.
3. Not 404 (route must exist).

## Errors

- Missing route 404; wrong shape.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.HTTPStatus == 404 || resp.HTTPStatus == 0 {
		t.Fatalf("bookmarks API missing; status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	if resp.HTTPStatus != 200 {
		t.Fatalf("status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	if !defaultRootOK(resp.Doc) {
		t.Fatalf("doc: %+v", resp.Doc)
	}
	if len(RootChildren(resp.Doc)) != 0 {
		t.Fatalf("want empty children")
	}
}
```
