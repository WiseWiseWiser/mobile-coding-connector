# Scenario

**Feature**: server error JSON surfaces in client error

```
# error message path
fake WS -> {"type":"error","message":"boom"} -> client error contains "boom"
```

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "ws-exec-error-message"
	req.WSErrorMessage = "boom"
	return nil
}
```