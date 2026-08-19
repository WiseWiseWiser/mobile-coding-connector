# Scenario

**Feature**: q quits without focus

```
ApplyKey("q") -> action Name=quit, no session id
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.ApplyKeys = []string{"q"}
	return nil
}
```
