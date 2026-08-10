# Scenario

**Feature**: status on unknown session id fails

```
agent-run status sess-nope -> Error: not found
```

## Steps

1. Args = `agent-run status sess-nope`; empty seeds.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "status", "sess-nope")
	req.Seeds = nil
	return nil
}
```
