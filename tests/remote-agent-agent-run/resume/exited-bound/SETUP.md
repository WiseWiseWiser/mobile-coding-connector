# Scenario

**Feature**: resume succeeds for exited + bound session (inject)

```
seedBoundExited + ResumeInject=ok -> agent-run resume sess-rb -> exit 0
```

## Steps

1. Seed finished with runner_session_id; ResumeInject `ok`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume", "sess-rb")
	req.Seeds = seedBoundExited("sess-rb", "grok-run-rb", "term-rb")
	req.ResumeInject = "ok"
	return nil
}
```
