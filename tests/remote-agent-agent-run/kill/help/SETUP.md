# Scenario

**Feature**: kill help

```
agent-run kill --help -> --dry-run
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "kill", "--help")
	return nil
}
```
