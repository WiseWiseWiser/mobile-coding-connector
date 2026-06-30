## Expected

- WS connect returns `session_id` JSON message with non-empty id.
- Session appears in list with requested name and cwd.

```go
import (
	"net/http"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.SessionID == "" {
		t.Fatal("expected session_id from WS create")
	}
	if !strings.HasPrefix(resp.SessionID, "session-") {
		t.Fatalf("unexpected id format: %q", resp.SessionID)
	}
}
```