# Scenario

**Feature**: resume without session id is rejected

```
agent-run resume  -> non-zero + clear error
```

## Steps

1. Args = `agent-run resume`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "resume")
	return nil
}
```
