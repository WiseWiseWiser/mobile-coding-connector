## Expected

1. Exit 0.
2. Soft: `ResumeCalled` true when inject is wired (preferred).
3. Does not print live-session refusal.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exited+bound resume should succeed; exit %d combined:\n%s", resp.ExitCode, resp.Combined)
	}
	msg := strings.ToLower(resp.Combined)
	if strings.Contains(msg, "is live") && strings.Contains(msg, "use send") {
		t.Fatalf("should not refuse as live; combined:\n%s", resp.Combined)
	}
	// Prefer inject observation when product wires ResumeSession.
	if req.ResumeInject == "ok" && !resp.ResumeCalled {
		// Soft until implementer wires inject end-to-end through CLI→API.
		// Fail hard only if stderr shows unknown subcommand.
		if strings.Contains(msg, "unknown") {
			t.Fatalf("resume not implemented; combined:\n%s", resp.Combined)
		}
	}
}
```
