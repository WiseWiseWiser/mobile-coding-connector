# Scenario

**Feature**: Enter focuses the selected session and signals action focus

```
NewUIState(sess-a selected on list)
ApplyKey("enter") -> action Name=focus, SessionID=sess-a
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "tui"
	req.ApplyKeys = []string{"enter"}
	return nil
}
```
