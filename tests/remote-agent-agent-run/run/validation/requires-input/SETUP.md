# Scenario

**Feature**: run without prompt/session when required fails

```
agent-run run  (no prompt, no --detach, no --session-id) -> error
```

Product may require a prompt for non-detach create; empty run must not hang.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Bare run with no flags/prompt — should fail validation, not call inject.
	setCLI(req, "agent-run", "run")
	return nil
}
```
