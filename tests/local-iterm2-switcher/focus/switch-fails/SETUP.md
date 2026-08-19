# Scenario

**Feature**: focus does not call Switch even when Switch would fail

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "focus"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.SessionID = "sess-a"
	req.SwitchErr = "not authorized"
	return nil
}
```
