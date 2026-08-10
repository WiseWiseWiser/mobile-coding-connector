# Scenario

**Feature**: resume --open invokes resume then attach path

```
seedBoundExited + ResumeInject=ok + WantAttachOnResume
  -> agent-run resume --open sess-open
  -> ResumeCalled && ResumeOpen && (AttachInvoked | attach-related success path)
```

## Preconditions

- L2 observes inject flags rather than a real interactive TTY attach.
- FakeAttach may be entered by server when open attach is requested.
- Non-interactive CLI may still surface attach TTY errors after resume; inject
  flags are the primary signal.

## Steps

1. Seed bound exited; ResumeInject ok; WantAttachOnResume; Args resume --open id.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume", "--open", "sess-open")
	req.Seeds = seedBoundExited("sess-open", "grok-run-open", "term-open")
	req.ResumeInject = "ok"
	req.WantAttachOnResume = true
	req.TTYMode = "hold"
	return nil
}
```
