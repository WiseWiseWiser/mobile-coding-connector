# Scenario

**Feature**: server error JSON surfaces in client error

```
# error message path
fake WS -> {"type":"error","message":"boom"} -> client error contains "boom"
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "ws-exec-error-message"
	req.WSErrorMessage = "boom"
	return nil
}
```