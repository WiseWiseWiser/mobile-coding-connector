# Scenario

**Feature**: Register mounts inventory, focus, notes

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "register"
	req.ITermRunning = true
	req.WindowSpace = 1
	return nil
}
```
