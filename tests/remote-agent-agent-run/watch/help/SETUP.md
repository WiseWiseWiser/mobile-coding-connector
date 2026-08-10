# Scenario

**Feature**: watch help

```
agent-run watch --help
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "watch", "--help")
	return nil
}
```
