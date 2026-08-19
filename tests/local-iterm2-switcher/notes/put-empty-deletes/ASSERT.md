## Expected

1. HTTP 200.
2. Store no longer has a note for sess-a.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !resp.OK {
		t.Fatalf("status=%d ok=%v body=%s", resp.StatusCode, resp.OK, resp.Body)
	}
	if resp.StoredNote != "" {
		t.Fatalf("stored=%q want deleted", resp.StoredNote)
	}
}
```
