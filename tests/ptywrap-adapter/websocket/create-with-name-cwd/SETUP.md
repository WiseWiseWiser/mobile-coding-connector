# Scenario

**Feature**: WS create-on-connect preserves ai-critic behavior

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "ws-create"
	req.SessionName = "adapter-compat"
	req.SessionCwd = t.TempDir()
	return nil
}
```