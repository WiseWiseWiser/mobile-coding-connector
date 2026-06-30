# Scenario

**Feature**: legacy WS terminal create via name and cwd query params

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "ws-create"
	return nil
}
```