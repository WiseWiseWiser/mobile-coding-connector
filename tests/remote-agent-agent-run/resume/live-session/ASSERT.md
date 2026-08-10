## Expected

1. Non-zero exit.
2. Error indicates session is live / running and suggests `send` or `attach`
   (or equivalent refuse-to-resume wording).
3. Soft: `ResumeCalled` true when inject is wired.

## Exit Code

non-zero

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
	if resp.ExitCode == 0 {
		t.Fatalf("live session resume must fail; stdout:\n%s", resp.Stdout)
	}
	msg := strings.ToLower(resp.Combined + "\n" + resp.Stderr)
	if !strings.Contains(msg, "live") && !strings.Contains(msg, "running") {
		t.Fatalf("expected live/running signal; combined:\n%s", resp.Combined)
	}
	if !strings.Contains(msg, "send") && !strings.Contains(msg, "attach") &&
		!strings.Contains(msg, "resume") {
		t.Fatalf("expected guidance away from resume (send/attach); combined:\n%s", resp.Combined)
	}
}
```
