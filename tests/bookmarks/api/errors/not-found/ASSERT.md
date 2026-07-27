## Expected

1. HTTPStatus == 404 (not 0/missing route for GET path alone — patch must be registered).
2. If status is 404 on missing route entirely, still fail distinctly when GET also 404.

## Errors

- 200 success; 500.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	// Missing implementation often yields 404 from httptest default — require BodyDoc GET works (API registered)
	if resp.Doc == nil || !defaultRootOK(resp.Doc) {
		t.Fatalf("bookmarks API not functional (GET tree); status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	if resp.HTTPStatus != 404 {
		t.Fatalf("want 404 for unknown id, got %d body=%s", resp.HTTPStatus, resp.Body)
	}
}
```
