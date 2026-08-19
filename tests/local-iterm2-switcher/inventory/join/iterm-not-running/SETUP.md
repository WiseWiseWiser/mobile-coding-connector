# Scenario

**Feature**: iTerm down still lists Desktops

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "join"
	req.ITermRunning = false
	return nil
}
```
