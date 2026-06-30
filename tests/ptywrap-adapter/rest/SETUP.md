# Scenario

**Feature**: REST list sessions API shape unchanged after refactor

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "list-shape"
	return nil
}
```