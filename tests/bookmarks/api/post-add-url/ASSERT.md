## Expected

1. HTTPStatus in 200–299.
2. Doc contains bm_docs or name Docs with correct url.

## Errors

- 404/500; node missing.

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
		t.Fatalf("API missing; status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	if resp.HTTPStatus < 200 || resp.HTTPStatus >= 300 {
		t.Fatalf("status=%d body=%s", resp.HTTPStatus, resp.Body)
	}
	n := FindNode(resp.Doc, "bm_docs")
	if n == nil {
		n = FindNodeByName(resp.Doc, "Docs")
	}
	if n == nil || n.URL != "https://docs.example.com" {
		t.Fatalf("node missing or wrong: doc=%+v body=%s", resp.Doc, resp.Body)
	}
}
```
