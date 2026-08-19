# Scenario

**Feature**: known session_id → Switch(2) then Focus

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
	return nil
}
```
