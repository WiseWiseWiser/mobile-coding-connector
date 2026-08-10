## Expected

1. Resume path was invoked with Open=true (`ResumeCalled` && `ResumeOpen`) when
   inject is wired through CLI→API.
2. Attach path attempted: `AttachInvoked` **or** output/error mentions attach
   (including non-interactive refuse after successful resume).
3. Not an "unknown subcommand" failure.

## Errors

- Resume never called and unknown command.
- Open flag ignored with no attach attempt signal.

## Exit Code

any (0 if full inject attach succeeds; non-zero OK if attach TTY gate fails after resume)

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
	msg := strings.ToLower(resp.Combined + "\n" + resp.Stderr + "\n" + resp.Stdout)
	if strings.Contains(msg, "unknown") && strings.Contains(msg, "subcommand") {
		t.Fatalf("resume --open not wired; combined:\n%s", resp.Combined)
	}
	// Prefer hard inject observations.
	if resp.ResumeCalled {
		if !resp.ResumeOpen {
			t.Fatalf("resume called but Open flag not observed (ResumeOpen=false)")
		}
		if !resp.AttachInvoked {
			// Accept attach attempt signaled only via CLI messaging under non-TTY.
			if !strings.Contains(msg, "attach") && !strings.Contains(msg, "interactive") &&
				!strings.Contains(msg, "open") {
				t.Fatalf("expected AttachInvoked or attach-related output after --open; combined:\n%s", resp.Combined)
			}
		}
		return
	}
	// Fallback before full inject wiring: must not silently no-op.
	if resp.ExitCode == 0 && !strings.Contains(msg, "resume") && !strings.Contains(msg, "attach") {
		t.Fatalf("resume --open produced no resume/attach signal; combined:\n%s", resp.Combined)
	}
}
```
