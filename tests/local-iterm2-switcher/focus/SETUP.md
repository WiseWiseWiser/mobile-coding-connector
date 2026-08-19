# Scenario

**Feature**: POST /api/local/iterm2/focus

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "focus"
	req.ITermRunning = true
	req.WindowSpace = 1
	return nil
}
```
