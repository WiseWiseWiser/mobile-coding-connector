# Scenario

**Feature**: GET inventory when iTerm is not running → 200, not 500

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = false
	return nil
}
```
