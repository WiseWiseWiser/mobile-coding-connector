## Expected

1. No error.
2. No stored note.
3. File does not exist.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StoredNote != "" {
		t.Fatalf("stored=%q", resp.StoredNote)
	}
	if resp.HasFile {
		t.Fatal("missing file should not be created on read")
	}
}
```
