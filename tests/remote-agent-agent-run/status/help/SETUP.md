# Scenario

**Feature**: status help

```
remote-agent agent-run status --help -> usage (session-id / home / json)
```

## Steps

1. Args = `agent-run status --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "status", "--help")
	return nil
}
```
