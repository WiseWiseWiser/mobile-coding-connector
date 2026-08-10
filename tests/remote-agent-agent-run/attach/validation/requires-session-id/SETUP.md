# Scenario

**Feature**: attach without session id is rejected

```
remote-agent agent-run attach  -> Error: requires session-id
```

## Preconditions

- No positional session id.

## Steps

1. Args = `agent-run attach` (no id).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "attach")
	return nil
}
```
