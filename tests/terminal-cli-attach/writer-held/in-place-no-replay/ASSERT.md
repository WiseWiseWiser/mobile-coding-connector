## Expected

- Typed echo still reaches the PTY (`MarkerInOutput`).
- Attach output does not contain `\e[?1049l` or `\e[2J` (`HasAltScreenReset == false`).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !resp.MarkerInOutput {
		t.Fatalf("typed echo %q must appear in attach output, got %q", req.Marker, resp.AttachOutput)
	}
	if resp.HasAltScreenReset {
		t.Fatalf("in-place attach must not send alt-screen reset or clear, got %q", resp.AttachOutput)
	}
}
```
