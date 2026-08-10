# Scenario

**Feature**: unknown subcommand under agent-run fails clearly

```
remote-agent agent-run nosuch -> Error: …, non-zero exit
```

## Preconditions

- `nosuch` is not a P1 (or later) agent-run subcommand.
- Product should reject with a clear message (unknown / usage), not hang.

## Steps

1. Args = `agent-run nosuch`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "nosuch")
	return nil
}
```
