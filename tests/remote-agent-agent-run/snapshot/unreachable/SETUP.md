# Scenario

**Feature**: snapshot error when TTY unreachable

```
SnapshotInject=error -> agent-run snapshot sess-x -> error
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "snapshot", "sess-x")
	req.Seeds = seedLiveTTY("sess-x", "term-x")
	req.SnapshotInject = "error"
	return nil
}
```
