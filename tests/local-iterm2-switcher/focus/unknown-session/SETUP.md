# Scenario

**Feature**: unknown session_id → 404, no Switch

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "focus"
	req.ITermRunning = true
	req.WindowSpace = 1
	req.SessionID = "sess-missing"
	return nil
}
```
