## Expected

- Session status is `exited` after the writer disconnects.
- `ErrIfSessionNotAttachable` returns an error containing `is exited`.
- The harness does not call `Attach` (`DidAttach == false`).

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.SessionStatus != "exited" {
		t.Fatalf("expected exited session, got %q", resp.SessionStatus)
	}
	if !strings.Contains(resp.AttachErr, "is exited") {
		t.Fatalf("expected attach gate error containing \"is exited\", got %q", resp.AttachErr)
	}
	if resp.DidAttach {
		t.Fatal("must not Attach to an exited session")
	}
}
```
