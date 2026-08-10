# Scenario

**Feature**: bare status prints remote agent-run home

```
agent-run status  -> home: <StoreHome>
```

## Preconditions

- No session id argument.
- Server store home is the Run-injected temp path (`resp.StoreHome`).

## Steps

1. Args = `agent-run status`; Seeds empty.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "status")
	req.Seeds = nil
	return nil
}
```
