# Scenario

**Feature**: msg rejects bad session/msg reference

```
agent-run msg status not-a-ref -> error
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "msg", "status", "not-a-ref")
	return nil
}
```
