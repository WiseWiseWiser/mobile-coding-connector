# Scenario

**Feature**: top-level agent-run help lists sessions

```
remote-agent agent-run --help -> Usage ... sessions ...
```

## Preconditions

None beyond L2 harness.

## Steps

1. Args = `agent-run --help`.

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
