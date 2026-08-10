# Scenario

**Feature**: top-level agent-run --help lists remote command tree

```
agent-run --help -> sessions attach status resume send msg snapshot watch kill run
```

Does not weaken sealed help/agent-run-root (sessions-only check remains).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "--help")
	return nil
}
```
