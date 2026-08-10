# Scenario

**Feature**: run help documents core flags

```
agent-run run --help
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "run", "--help")
	return nil
}
```
