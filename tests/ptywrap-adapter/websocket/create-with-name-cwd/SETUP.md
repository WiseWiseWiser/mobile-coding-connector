# Scenario

**Feature**: WS create-on-connect preserves ai-critic behavior

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "ws-create"
	req.SessionName = "adapter-compat"
	req.SessionCwd = t.TempDir()
	return nil
}
```