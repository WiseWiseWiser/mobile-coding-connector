# Scenario

**Feature**: legacy WS terminal create via name and cwd query params

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "ws-create"
	return nil
}
```