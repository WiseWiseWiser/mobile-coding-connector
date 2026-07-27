# Scenario

**Feature**: HTTP dial failure includes status and JSON body snippet

```
# dial error path (terminalDialError behavior)
HTTP 401 + JSON body -> WS dial fails -> error mentions status and body
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "ws-dial-http-error"
	req.WSDialHTTPStatus = 401
	req.WSDialHTTPBody = `{"error":"unauthorized"}`
	return nil
}
```