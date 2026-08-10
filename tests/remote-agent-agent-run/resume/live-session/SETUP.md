# Scenario

**Feature**: resume rejects live (still running) sessions

```
seed live+bound + ResumeInject=error-live
  -> agent-run resume sess-live -> error (send/attach)
```

## Steps

1. Seed live bound session; ResumeInject `error-live`; CLI resume.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume", "sess-live")
	req.Seeds = seedLiveBound("sess-live", "grok-run-live")
	req.ResumeInject = "error-live"
	return nil
}
```
