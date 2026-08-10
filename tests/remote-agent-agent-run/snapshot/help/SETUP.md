# Scenario

**Feature**: snapshot help

```
agent-run snapshot --help
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "snapshot", "--help")
	return nil
}
```
