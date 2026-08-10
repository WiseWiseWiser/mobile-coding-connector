# Scenario

**Feature**: agent-run sessions help documents list flags

```
remote-agent agent-run sessions --help -> --json --limit
```

## Preconditions

None beyond L2 harness.

## Steps

1. Args = `agent-run sessions --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions", "--help")
	return nil
}
```
