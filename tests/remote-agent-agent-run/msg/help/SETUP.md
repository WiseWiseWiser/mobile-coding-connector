# Scenario

**Feature**: msg help

```
agent-run msg --help -> status|cancel
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "msg", "--help")
	return nil
}
```
