## Expected

1. HTTPStatus 400 (4xx validation).
2. GET tree still only default root (no empty-name node).

## Errors

- 2xx storing invalid node.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.HTTPStatus == 404 && (resp.Doc == nil || !defaultRootOK(resp.Doc)) {
		t.Fatalf("API missing; status=%d", resp.HTTPStatus)
	}
	if resp.HTTPStatus < 400 || resp.HTTPStatus >= 500 {
		t.Fatalf("want 4xx validation, got %d body=%s", resp.HTTPStatus, resp.Body)
	}
	if len(RootChildren(resp.Doc)) != 0 {
		t.Fatalf("invalid node should not be stored: %+v", RootChildren(resp.Doc))
	}
}
```
