## Expected

Desired product behavior (inverted from the crime scene):

- Attach produced PTY output (prompt / scrollback). The connection did not
  time out or fail before the snapshot.
- Typed `echo CLI_ATTACH_MARKER` ran: `resp.MarkerInOutput == true`.
- Session is still `running` and `WriterConnected` is still true (CLI attach
  did not reap the shell or steal the writer away from the held socket).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.SessionID == "" {
		t.Fatal("expected a session id")
	}
	if resp.AttachOutput == "" {
		t.Fatal("attach produced no PTY output")
	}
	if !resp.MarkerInOutput {
		t.Fatalf("typed echo %q must appear in attach output, got %q", req.Marker, resp.AttachOutput)
	}
	if resp.SessionStatus != "running" {
		t.Fatalf("session must stay running after CLI attach types, got status %q", resp.SessionStatus)
	}
	if !resp.WriterConnected {
		t.Fatal("original writer must stay connected after CLI attach types")
	}
}
```
