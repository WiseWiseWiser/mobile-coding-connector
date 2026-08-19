## Expected

1. HTTP 200 ok.
2. Store has the note.
3. File exists.

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
	if resp.StoredNote != "fix auth on staging" {
		t.Fatalf("stored=%q", resp.StoredNote)
	}
	if !resp.HasFile {
		t.Fatal("want notes file written")
	}
}
```
