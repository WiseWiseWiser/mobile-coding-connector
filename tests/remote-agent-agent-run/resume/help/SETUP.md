# Scenario

**Feature**: resume help

```
remote-agent agent-run resume --help -> session-id, --open
```

## Steps

1. Args = `agent-run resume --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume", "--help")
	return nil
}
```
