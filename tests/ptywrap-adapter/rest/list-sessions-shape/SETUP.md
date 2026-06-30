# Scenario

**Feature**: GET /api/terminal/sessions JSON schema regression

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Phase = "list-shape"
	return nil
}
```