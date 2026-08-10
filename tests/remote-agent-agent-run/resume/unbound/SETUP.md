# Scenario

**Feature**: resume fails without runner_session_id bind

```
seed finished without runner_session_id + ResumeInject=error-unbound
  -> agent-run resume sess-ub -> error
```

## Steps

1. Seed unbound finished; ResumeInject `error-unbound`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume", "sess-ub")
	req.Seeds = seedUnboundFinished("sess-ub")
	req.ResumeInject = "error-unbound"
	return nil
}
```
